// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stackdriver

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"reflect"
	"strconv"
	"time"

	gce "cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	sd_api "google.golang.org/api/monitoring/v3"
	gce_util "k8s.io/heapster/common/gce"
	"k8s.io/heapster/metrics/core"
	"strings"
)

const (
	maxTimeseriesPerRequest = 200
	// 2 seconds on SD side, 1 extra for networking overhead
	sdRequestLatencySec           = 3
	httpResponseCodeUnknown       = -100
	httpResponseCodeClientTimeout = -1
)

type StackdriverSink struct {
	project               string
	cluster               string
	zone                  string
	stackdriverClient     *sd_api.Service
	minInterval           time.Duration
	lastExportTime        time.Time
	batchExportTimeoutSec int
	initialDelaySec       int
	useOldResourceModel   bool
	useNewResourceModel   bool
}

type metricMetadata struct {
	MetricKind string
	ValueType  string
	Name       string
}

var (
	// Sink performance metrics

	requestsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "heapster",
			Subsystem: "stackdriver",
			Name:      "requests_count",
			Help:      "Number of requests with return codes",
		},
		[]string{"code"},
	)

	timeseriesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "heapster",
			Subsystem: "stackdriver",
			Name:      "timeseries_count",
			Help:      "Number of Timeseries sent with return codes",
		},
		[]string{"code"},
	)
	requestLatency = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: "heapster",
			Subsystem: "stackdriver",
			Name:      "request_latency_milliseconds",
			Help:      "Latency of requests to Stackdriver Monitoring API.",
		},
	)
)

func (sink *StackdriverSink) Name() string {
	return "Stackdriver Sink"
}

func (sink *StackdriverSink) Stop() {
}

func (sink *StackdriverSink) processMetrics(metricValues map[string]core.MetricValue,
	timestamp time.Time, labels map[string]string, collectionStartTime time.Time) []*sd_api.TimeSeries {
	timeseries := make([]*sd_api.TimeSeries, 0)
	if sink.useOldResourceModel {
		for name, value := range metricValues {
			if ts := sink.LegacyTranslateMetric(timestamp, labels, name, value, collectionStartTime); ts != nil {
				timeseries = append(timeseries, ts)
			}
		}
	}
	if sink.useNewResourceModel {
		for name, value := range metricValues {
			if ts := sink.TranslateMetric(timestamp, labels, name, value, collectionStartTime); ts != nil {
				timeseries = append(timeseries, ts)
			}
		}
	}
	return timeseries
}

func (sink *StackdriverSink) ExportData(dataBatch *core.DataBatch) {
	// Make sure we don't export metrics too often.
	if dataBatch.Timestamp.Before(sink.lastExportTime.Add(sink.minInterval)) {
		glog.V(2).Infof("Skipping batch from %s because there hasn't passed %s from last export time %s", dataBatch.Timestamp, sink.minInterval, sink.lastExportTime)
		return
	}
	sink.lastExportTime = dataBatch.Timestamp

	requests := []*sd_api.CreateTimeSeriesRequest{}
	req := getReq()
	for key, metricSet := range dataBatch.MetricSets {
		switch metricSet.Labels["type"] {
		case core.MetricSetTypeNode, core.MetricSetTypePod, core.MetricSetTypePodContainer, core.MetricSetTypeSystemContainer:
		default:
			continue
		}

		if metricSet.CollectionStartTime.IsZero() {
			glog.V(2).Infof("Skipping incorrect metric set %s because collection start time is zero", key)
			continue
		}

		// Hack used with legacy resource type "gke_container". It is used to represent three
		// Kubernetes resources: container, pod or node. For pods container name is empty, for nodes it
		// is set to artificial value "machine". Otherwise it stores actual container name.
		// With new resource types, container_name is ignored for resources other than "k8s_container"
		if sink.useOldResourceModel && metricSet.Labels["type"] == core.MetricSetTypeNode {
			metricSet.Labels[core.LabelContainerName.Key] = "machine"
		}

		derivedMetrics := sink.computeDerivedMetrics(metricSet)

		derivedTimeseries := sink.processMetrics(derivedMetrics.MetricValues, dataBatch.Timestamp, metricSet.Labels, metricSet.CollectionStartTime)
		timeseries := sink.processMetrics(metricSet.MetricValues, dataBatch.Timestamp, metricSet.Labels, metricSet.CollectionStartTime)

		timeseries = append(timeseries, derivedTimeseries...)

		for _, ts := range timeseries {
			req.TimeSeries = append(req.TimeSeries, ts)
			if len(req.TimeSeries) >= maxTimeseriesPerRequest {
				requests = append(requests, req)
				req = getReq()
			}
		}

		for _, metric := range metricSet.LabeledMetrics {
			if sink.useOldResourceModel {
				if point := sink.LegacyTranslateLabeledMetric(dataBatch.Timestamp, metricSet.Labels, metric, metricSet.CollectionStartTime); point != nil {
					req.TimeSeries = append(req.TimeSeries, point)
				}

				if len(req.TimeSeries) >= maxTimeseriesPerRequest {
					requests = append(requests, req)
					req = getReq()
				}
			}
			if sink.useNewResourceModel {
				point := sink.TranslateLabeledMetric(dataBatch.Timestamp, metricSet.Labels, metric, metricSet.CollectionStartTime)
				if point != nil {
					req.TimeSeries = append(req.TimeSeries, point)
				}

				if len(req.TimeSeries) >= maxTimeseriesPerRequest {
					requests = append(requests, req)
					req = getReq()
				}
			}
		}
	}

	if len(req.TimeSeries) > 0 {
		requests = append(requests, req)
	}

	go sink.sendRequests(requests)
}

func (sink *StackdriverSink) sendRequests(requests []*sd_api.CreateTimeSeriesRequest) {
	// Each worker can handle at least batchExportTimeout/sdRequestLatencySec requests within the specified period.
	// 5 extra workers just in case.
	workers := 5 + len(requests)/(sink.batchExportTimeoutSec/sdRequestLatencySec)
	requestQueue := make(chan *sd_api.CreateTimeSeriesRequest)
	completedQueue := make(chan bool)

	// Launch Go routines responsible for sending requests
	for i := 0; i < workers; i++ {
		go sink.requestSender(requestQueue, completedQueue)
	}

	timeout := time.Duration(sink.batchExportTimeoutSec) * time.Second
	timeoutSending := time.After(timeout)
	timeoutCompleted := time.After(timeout)

forloop:
	for i, r := range requests {
		select {
		case requestQueue <- r:
			// yet another request added to queue
		case <-timeoutSending:
			glog.Warningf("Timeout while exporting metrics to Stackdriver. Dropping %d out of %d requests.", len(requests)-i, len(requests))
			// TODO(piosz): consider cancelling requests in flight
			// Report dropped requests in metrics.
			for _, req := range requests[i:] {
				requestsSent.WithLabelValues(strconv.Itoa(httpResponseCodeClientTimeout)).Inc()
				timeseriesSent.
					WithLabelValues(strconv.Itoa(httpResponseCodeClientTimeout)).
					Add(float64(len(req.TimeSeries)))
			}
			break forloop
		}
	}

	// Close the channel in order to cancel exporting routines.
	close(requestQueue)

	workersCompleted := 0
	for {
		select {
		case <-completedQueue:
			workersCompleted++
			if workersCompleted == workers {
				glog.V(4).Infof("All %d workers successfully finished sending requests to SD.", workersCompleted)
				return
			}
		case <-timeoutCompleted:
			glog.Warningf("Only %d out of %d workers successfully finished sending requests to SD. Some metrics might be lost.", workersCompleted, workers)
			return
		}
	}
}

func (sink *StackdriverSink) requestSender(reqQueue chan *sd_api.CreateTimeSeriesRequest, completedQueue chan bool) {
	defer func() {
		completedQueue <- true
	}()
	time.Sleep(time.Duration(rand.Intn(1000*sink.initialDelaySec)) * time.Millisecond)
	for {
		select {
		case req, active := <-reqQueue:
			if !active {
				return
			}
			sink.sendOneRequest(req)
		}
	}
}

func marshalRequestAndLog(printer func([]byte), req *sd_api.CreateTimeSeriesRequest) {
	reqJson, errJson := json.Marshal(req)
	if errJson != nil {
		glog.Errorf("Couldn't marshal Stackdriver request %v", errJson)
	} else {
		printer(reqJson)
	}
}

func (sink *StackdriverSink) sendOneRequest(req *sd_api.CreateTimeSeriesRequest) {
	startTime := time.Now()
	empty, err := sink.stackdriverClient.Projects.TimeSeries.Create(fullProjectName(sink.project), req).Do()

	var responseCode int
	if err != nil {
		glog.Warningf("Error while sending request to Stackdriver %v", err)
		// Convert request to json and log it, but only if logging level is equal to 2 or more.
		if glog.V(2) {
			marshalRequestAndLog(func(reqJson []byte) {
				glog.V(2).Infof("The request was: %s", reqJson)
			}, req)
		}
		switch reflect.Indirect(reflect.ValueOf(err)).Type() {
		case reflect.Indirect(reflect.ValueOf(&googleapi.Error{})).Type():
			responseCode = err.(*googleapi.Error).Code
		default:
			responseCode = httpResponseCodeUnknown
		}
	} else {
		// Convert request to json and log it, but only if logging level is equal to 10 or more.
		if glog.V(10) {
			marshalRequestAndLog(func(reqJson []byte) {
				glog.V(10).Infof("Stackdriver request sent: %s", reqJson)
			}, req)
		}
		responseCode = empty.ServerResponse.HTTPStatusCode
	}

	requestsSent.WithLabelValues(strconv.Itoa(responseCode)).Inc()
	timeseriesSent.
		WithLabelValues(strconv.Itoa(responseCode)).
		Add(float64(len(req.TimeSeries)))
	requestLatency.Observe(time.Since(startTime).Seconds() / time.Millisecond.Seconds())
}

func CreateStackdriverSink(uri *url.URL) (core.DataSink, error) {
	if len(uri.Scheme) > 0 {
		return nil, fmt.Errorf("Scheme should not be set for Stackdriver sink")
	}
	if len(uri.Host) > 0 {
		return nil, fmt.Errorf("Host should not be set for Stackdriver sink")
	}

	opts := uri.Query()

	useOldResourceModel := true
	if err := parseBoolFlag(opts, "use_old_resources", &useOldResourceModel); err != nil {
		return nil, err
	}
	useNewResourceModel := false
	if err := parseBoolFlag(opts, "use_new_resources", &useNewResourceModel); err != nil {
		return nil, err
	}

	cluster_name := ""
	if len(opts["cluster_name"]) >= 1 {
		cluster_name = opts["cluster_name"][0]
	}

	minInterval := time.Nanosecond
	if len(opts["min_interval_sec"]) >= 1 {
		if interval, err := strconv.Atoi(opts["min_interval_sec"][0]); err != nil {
			return nil, fmt.Errorf("Min interval should be an integer, found: %v", opts["min_interval_sec"][0])
		} else {
			minInterval = time.Duration(interval) * time.Second
		}
	}

	batchExportTimeoutSec := 60
	var err error
	if len(opts["batch_export_timeout_sec"]) >= 1 {
		if batchExportTimeoutSec, err = strconv.Atoi(opts["batch_export_timeout_sec"][0]); err != nil {
			return nil, fmt.Errorf("Batch export timeout should be an integer, found: %v", opts["batch_export_timeout_sec"][0])
		}
	}

	initialDelaySec := sdRequestLatencySec
	if len(opts["initial_delay_sec"]) >= 1 {
		if initialDelaySec, err = strconv.Atoi(opts["initial_delay_sec"][0]); err != nil {
			return nil, fmt.Errorf("Initial delay should be an integer, found: %v", opts["initial_delay_sec"][0])
		}
	}

	if err := gce_util.EnsureOnGCE(); err != nil {
		return nil, err
	}

	// Detect project ID
	projectId, err := gce.ProjectID()
	if err != nil {
		return nil, err
	}

	// Detect zone
	zone, err := gce.Zone()
	if err != nil {
		return nil, err
	}

	// Create Google Cloud Monitoring service
	client := oauth2.NewClient(oauth2.NoContext, google.ComputeTokenSource(""))
	stackdriverClient, err := sd_api.New(client)
	if err != nil {
		return nil, err
	}

	sink := &StackdriverSink{
		project:               projectId,
		cluster:               cluster_name,
		zone:                  zone,
		stackdriverClient:     stackdriverClient,
		minInterval:           minInterval,
		batchExportTimeoutSec: batchExportTimeoutSec,
		initialDelaySec:       initialDelaySec,
		useOldResourceModel:   useOldResourceModel,
		useNewResourceModel:   useNewResourceModel,
	}

	// Register sink metrics
	prometheus.MustRegister(requestsSent)
	prometheus.MustRegister(timeseriesSent)
	prometheus.MustRegister(requestLatency)

	glog.Infof("Created Stackdriver sink")

	return sink, nil
}

func parseBoolFlag(opts map[string][]string, name string, targetValue *bool) error {
	if len(opts[name]) >= 1 {
		var err error
		*targetValue, err = strconv.ParseBool(opts[name][0])
		if err != nil {
			return fmt.Errorf("%s = %s is not correct boolean value", name, opts[name][0])
		}
	}
	return nil
}

func (sink *StackdriverSink) computeDerivedMetrics(metricSet *core.MetricSet) *core.MetricSet {
	newMetricSet := &core.MetricSet{MetricValues: map[string]core.MetricValue{}}
	usage, usageOK := metricSet.MetricValues[core.MetricMemoryUsage.MetricDescriptor.Name]
	workingSet, workingSetOK := metricSet.MetricValues[core.MetricMemoryWorkingSet.MetricDescriptor.Name]

	if usageOK && workingSetOK {
		newMetricSet.MetricValues["memory/bytes_used"] = core.MetricValue{
			IntValue: usage.IntValue - workingSet.IntValue,
		}
	}

	memoryFaults, memoryFaultsOK := metricSet.MetricValues[core.MetricMemoryPageFaults.MetricDescriptor.Name]
	majorMemoryFaults, majorMemoryFaultsOK := metricSet.MetricValues[core.MetricMemoryMajorPageFaults.MetricDescriptor.Name]
	if memoryFaultsOK && majorMemoryFaultsOK {
		newMetricSet.MetricValues["memory/minor_page_faults"] = core.MetricValue{
			IntValue: memoryFaults.IntValue - majorMemoryFaults.IntValue,
		}
	}

	return newMetricSet
}

func (sink *StackdriverSink) LegacyTranslateLabeledMetric(timestamp time.Time, labels map[string]string, metric core.LabeledMetric, collectionStartTime time.Time) *sd_api.TimeSeries {
	resourceLabels := sink.legacyGetResourceLabels(labels)
	switch metric.Name {
	case core.MetricFilesystemUsage.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, timestamp, metric.IntValue)
		ts := legacyCreateTimeSeries(resourceLabels, legacyDiskBytesUsedMD, point)
		ts.Metric.Labels = map[string]string{
			"device_name": metric.Labels[core.LabelResourceID.Key],
		}
		return ts
	case core.MetricFilesystemLimit.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, timestamp, metric.IntValue)
		ts := legacyCreateTimeSeries(resourceLabels, legacyDiskBytesTotalMD, point)
		ts.Metric.Labels = map[string]string{
			"device_name": metric.Labels[core.LabelResourceID.Key],
		}
		return ts
	}
	return nil
}

func (sink *StackdriverSink) LegacyTranslateMetric(timestamp time.Time, labels map[string]string, name string, value core.MetricValue, collectionStartTime time.Time) *sd_api.TimeSeries {
	resourceLabels := sink.legacyGetResourceLabels(labels)
	if !collectionStartTime.Before(timestamp) {
		glog.V(4).Infof("Error translating metric %v for pod %v: batch timestamp %v earlier than pod create time %v", name, labels["pod_name"], timestamp, collectionStartTime)
		return nil
	}
	switch name {
	case core.MetricUptime.MetricDescriptor.Name:
		doubleValue := float64(value.IntValue) / float64(time.Second/time.Millisecond)
		point := sink.doublePoint(timestamp, collectionStartTime, doubleValue)
		return legacyCreateTimeSeries(resourceLabels, legacyUptimeMD, point)
	case core.MetricCpuLimit.MetricDescriptor.Name:
		// converting from millicores to cores
		point := sink.doublePoint(timestamp, timestamp, float64(value.IntValue)/1000)
		return legacyCreateTimeSeries(resourceLabels, legacyCPUReservedCoresMD, point)
	case core.MetricCpuUsage.MetricDescriptor.Name:
		point := sink.doublePoint(timestamp, collectionStartTime, float64(value.IntValue)/float64(time.Second/time.Nanosecond))
		return legacyCreateTimeSeries(resourceLabels, legacyCPUUsageTimeMD, point)
	case core.MetricNetworkRx.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
		return legacyCreateTimeSeries(resourceLabels, legacyNetworkRxMD, point)
	case core.MetricNetworkTx.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
		return legacyCreateTimeSeries(resourceLabels, legacyNetworkTxMD, point)
	case core.MetricMemoryLimit.MetricDescriptor.Name:
		// omit nodes, using memory/node_allocatable instead
		if labels["type"] == core.MetricSetTypeNode {
			return nil
		}
		point := sink.intPoint(timestamp, timestamp, value.IntValue)
		return legacyCreateTimeSeries(resourceLabels, legacyMemoryLimitMD, point)
	case core.MetricNodeMemoryAllocatable.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, timestamp, value.IntValue)
		return legacyCreateTimeSeries(resourceLabels, legacyMemoryLimitMD, point)
	case core.MetricMemoryMajorPageFaults.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
		ts := legacyCreateTimeSeries(resourceLabels, legacyMemoryPageFaultsMD, point)
		ts.Metric.Labels = map[string]string{
			"fault_type": "major",
		}
		return ts
	case "memory/bytes_used":
		point := sink.intPoint(timestamp, timestamp, value.IntValue)
		ts := legacyCreateTimeSeries(resourceLabels, legacyMemoryBytesUsedMD, point)
		ts.Metric.Labels = map[string]string{
			"memory_type": "evictable",
		}
		return ts
	case core.MetricMemoryWorkingSet.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, timestamp, value.IntValue)
		ts := legacyCreateTimeSeries(resourceLabels, legacyMemoryBytesUsedMD, point)
		ts.Metric.Labels = map[string]string{
			"memory_type": "non-evictable",
		}
		return ts
	case "memory/minor_page_faults":
		point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
		ts := legacyCreateTimeSeries(resourceLabels, legacyMemoryPageFaultsMD, point)
		ts.Metric.Labels = map[string]string{
			"fault_type": "minor",
		}
		return ts
	}
	return nil
}

func (sink *StackdriverSink) TranslateLabeledMetric(timestamp time.Time, labels map[string]string, metric core.LabeledMetric, collectionStartTime time.Time) *sd_api.TimeSeries {
	switch labels["type"] {
	case core.MetricSetTypePod:
		podLabels := sink.getPodResourceLabels(labels)
		switch metric.Name {
		case core.MetricFilesystemUsage.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, metric.MetricValue.IntValue)
			ts := createTimeSeries("k8s_pod", podLabels, volumeUsedBytesMD, point)
			ts.Metric.Labels = map[string]string{
				core.LabelVolumeName.Key: strings.TrimPrefix(metric.Labels[core.LabelResourceID.Key], "Volume:"),
			}
			return ts
		case core.MetricFilesystemLimit.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, metric.MetricValue.IntValue)
			ts := createTimeSeries("k8s_pod", podLabels, volumeTotalBytesMD, point)
			ts.Metric.Labels = map[string]string{
				core.LabelVolumeName.Key: strings.TrimPrefix(metric.Labels[core.LabelResourceID.Key], "Volume:"),
			}
			return ts
		}
	}
	return nil
}

func (sink *StackdriverSink) TranslateMetric(timestamp time.Time, labels map[string]string, name string, value core.MetricValue, collectionStartTime time.Time) *sd_api.TimeSeries {
	if !collectionStartTime.Before(timestamp) {
		glog.V(4).Infof("Error translating metric %v for pod %v: batch timestamp %v earlier than pod create time %v", name, labels["pod_name"], timestamp, collectionStartTime)
		return nil
	}
	switch labels["type"] {
	case core.MetricSetTypePodContainer:
		containerLabels := sink.getContainerResourceLabels(labels)
		switch name {
		case core.MetricUptime.MetricDescriptor.Name:
			doubleValue := float64(value.IntValue) / float64(time.Second/time.Millisecond)
			point := sink.doublePoint(timestamp, timestamp, doubleValue)
			return createTimeSeries("k8s_container", containerLabels, containerUptimeMD, point)
		case core.MetricCpuLimit.MetricDescriptor.Name:
			point := sink.doublePoint(timestamp, timestamp, float64(value.IntValue)/1000)
			return createTimeSeries("k8s_container", containerLabels, cpuLimitCoresMD, point)
		case core.MetricCpuRequest.MetricDescriptor.Name:
			point := sink.doublePoint(timestamp, timestamp, float64(value.IntValue)/1000)
			return createTimeSeries("k8s_container", containerLabels, cpuRequestedCoresMD, point)
		case core.MetricCpuUsage.MetricDescriptor.Name:
			point := sink.doublePoint(timestamp, collectionStartTime, float64(value.IntValue)/float64(time.Second/time.Nanosecond))
			return createTimeSeries("k8s_container", containerLabels, cpuContainerCoreUsageTimeMD, point)
		case core.MetricMemoryLimit.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			return createTimeSeries("k8s_container", containerLabels, memoryLimitBytesMD, point)
		case "memory/bytes_used":
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			ts := createTimeSeries("k8s_container", containerLabels, memoryContainerUsedBytesMD, point)
			ts.Metric.Labels = map[string]string{
				"memory_type": "evictable",
			}
			return ts
		case core.MetricMemoryWorkingSet.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			ts := createTimeSeries("k8s_container", containerLabels, memoryContainerUsedBytesMD, point)
			ts.Metric.Labels = map[string]string{
				"memory_type": "non-evictable",
			}
			return ts
		case core.MetricMemoryRequest.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			return createTimeSeries("k8s_container", containerLabels, memoryRequestedBytesMD, point)
		case core.MetricRestartCount.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			return createTimeSeries("k8s_container", containerLabels, restartCountMD, point)
		}
	case core.MetricSetTypePod:
		podLabels := sink.getPodResourceLabels(labels)
		switch name {
		case core.MetricNetworkRx.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
			return createTimeSeries("k8s_pod", podLabels, networkPodReceivedBytesMD, point)
		case core.MetricNetworkTx.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
			return createTimeSeries("k8s_pod", podLabels, networkPodSentBytesMD, point)
		}
	case core.MetricSetTypeNode:
		nodeLabels := sink.getNodeResourceLabels(labels)
		switch name {
		case core.MetricNodeCpuCapacity.MetricDescriptor.Name:
			point := sink.doublePoint(timestamp, timestamp, float64(value.FloatValue))
			return createTimeSeries("k8s_node", nodeLabels, cpuTotalCoresMD, point)
		case core.MetricNodeCpuAllocatable.MetricDescriptor.Name:
			point := sink.doublePoint(timestamp, timestamp, float64(value.FloatValue))
			return createTimeSeries("k8s_node", nodeLabels, cpuAllocatableCoresMD, point)
		case core.MetricCpuUsage.MetricDescriptor.Name:
			point := sink.doublePoint(timestamp, collectionStartTime, float64(value.IntValue)/float64(time.Second/time.Nanosecond))
			return createTimeSeries("k8s_node", nodeLabels, cpuNodeCoreUsageTimeMD, point)
		case core.MetricNodeMemoryCapacity.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			return createTimeSeries("k8s_node", nodeLabels, memoryTotalBytesMD, point)
		case core.MetricNodeMemoryAllocatable.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			return createTimeSeries("k8s_node", nodeLabels, memoryAllocatableBytesMD, point)
		case "memory/bytes_used":
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			ts := createTimeSeries("k8s_node", nodeLabels, memoryNodeUsedBytesMD, point)
			ts.Metric.Labels = map[string]string{
				"memory_type": "evictable",
			}
			return ts
		case core.MetricMemoryWorkingSet.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			ts := createTimeSeries("k8s_node", nodeLabels, memoryNodeUsedBytesMD, point)
			ts.Metric.Labels = map[string]string{
				"memory_type": "non-evictable",
			}
			return ts
		case core.MetricNetworkRx.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
			return createTimeSeries("k8s_node", nodeLabels, networkNodeReceivedBytesMD, point)
		case core.MetricNetworkTx.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, collectionStartTime, value.IntValue)
			return createTimeSeries("k8s_node", nodeLabels, networkNodeSentBytesMD, point)
		}
	case core.MetricSetTypeSystemContainer:
		nodeLabels := sink.getNodeResourceLabels(labels)
		switch name {
		case "memory/bytes_used":
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			ts := createTimeSeries("k8s_node", nodeLabels, memoryNodeDaemonUsedBytesMD, point)
			ts.Metric.Labels = map[string]string{
				"component":   labels[core.LabelContainerName.Key],
				"memory_type": "evictable",
			}
			return ts
		case core.MetricMemoryWorkingSet.MetricDescriptor.Name:
			point := sink.intPoint(timestamp, timestamp, value.IntValue)
			ts := createTimeSeries("k8s_node", nodeLabels, memoryNodeDaemonUsedBytesMD, point)
			ts.Metric.Labels = map[string]string{
				"component":   labels[core.LabelContainerName.Key],
				"memory_type": "non-evictable",
			}
			return ts
		case core.MetricCpuUsage.MetricDescriptor.Name:
			point := sink.doublePoint(timestamp, collectionStartTime, float64(value.IntValue)/float64(time.Second/time.Nanosecond))
			ts := createTimeSeries("k8s_node", nodeLabels, cpuNodeDaemonCoreUsageTimeMD, point)
			ts.Metric.Labels = map[string]string{
				"component": labels[core.LabelContainerName.Key],
			}
			return ts
		}
	}
	return nil
}

func (sink *StackdriverSink) legacyGetResourceLabels(labels map[string]string) map[string]string {
	return map[string]string{
		"project_id":     sink.project,
		"cluster_name":   sink.cluster,
		"zone":           sink.zone, // TODO(kawych): revisit how the location is set
		"instance_id":    labels[core.LabelHostID.Key],
		"namespace_id":   labels[core.LabelPodNamespaceUID.Key],
		"pod_id":         labels[core.LabelPodId.Key],
		"container_name": labels[core.LabelContainerName.Key],
	}
}

func (sink *StackdriverSink) getContainerResourceLabels(labels map[string]string) map[string]string {
	return map[string]string{
		"project_id":     sink.project,
		"location":       sink.zone, // TODO(kawych): revisit how the location is set
		"cluster_name":   sink.cluster,
		"namespace_name": labels[core.LabelNamespaceName.Key],
		"node_name":      labels[core.LabelNodename.Key],
		"pod_name":       labels[core.LabelPodName.Key],
		"container_name": labels[core.LabelContainerName.Key],
	}
}

func (sink *StackdriverSink) getPodResourceLabels(labels map[string]string) map[string]string {
	return map[string]string{
		"project_id":     sink.project,
		"location":       sink.zone, // TODO(kawych): revisit how the location is set
		"cluster_name":   sink.cluster,
		"namespace_name": labels[core.LabelNamespaceName.Key],
		"node_name":      labels[core.LabelNodename.Key],
		"pod_name":       labels[core.LabelPodName.Key],
	}
}

func (sink *StackdriverSink) getNodeResourceLabels(labels map[string]string) map[string]string {
	return map[string]string{
		"project_id":   sink.project,
		"location":     sink.zone, // TODO(kawych): revisit how the location is set
		"cluster_name": sink.cluster,
		"node_name":    labels[core.LabelNodename.Key],
	}
}

func legacyCreateTimeSeries(resourceLabels map[string]string, metadata *metricMetadata, point *sd_api.Point) *sd_api.TimeSeries {
	return createTimeSeries("gke_container", resourceLabels, metadata, point)
}

func createTimeSeries(resource string, resourceLabels map[string]string, metadata *metricMetadata, point *sd_api.Point) *sd_api.TimeSeries {
	return &sd_api.TimeSeries{
		Metric: &sd_api.Metric{
			Type: metadata.Name,
		},
		MetricKind: metadata.MetricKind,
		ValueType:  metadata.ValueType,
		Resource: &sd_api.MonitoredResource{
			Labels: resourceLabels,
			Type:   resource,
		},
		Points: []*sd_api.Point{point},
	}
}

func (sink *StackdriverSink) doublePoint(endTime time.Time, startTime time.Time, value float64) *sd_api.Point {
	return &sd_api.Point{
		Interval: &sd_api.TimeInterval{
			EndTime:   endTime.Format(time.RFC3339),
			StartTime: startTime.Format(time.RFC3339),
		},
		Value: &sd_api.TypedValue{
			DoubleValue:     &value,
			ForceSendFields: []string{"DoubleValue"},
		},
	}

}

func (sink *StackdriverSink) intPoint(endTime time.Time, startTime time.Time, value int64) *sd_api.Point {
	return &sd_api.Point{
		Interval: &sd_api.TimeInterval{
			EndTime:   endTime.Format(time.RFC3339),
			StartTime: startTime.Format(time.RFC3339),
		},
		Value: &sd_api.TypedValue{
			Int64Value:      &value,
			ForceSendFields: []string{"Int64Value"},
		},
	}
}

func fullProjectName(name string) string {
	return fmt.Sprintf("projects/%s", name)
}

func getReq() *sd_api.CreateTimeSeriesRequest {
	return &sd_api.CreateTimeSeriesRequest{TimeSeries: make([]*sd_api.TimeSeries, 0)}
}
