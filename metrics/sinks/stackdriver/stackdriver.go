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
	"fmt"
	"net/url"
	"time"

	gce "cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sd_api "google.golang.org/api/monitoring/v3"
	gce_util "k8s.io/heapster/common/gce"
	"k8s.io/heapster/metrics/core"
)

const (
	maxTimeseriesPerRequest = 200
)

type stackdriverSink struct {
	project           string
	zone              string
	stackdriverClient *sd_api.Service
}

type metricMetadata struct {
	MetricKind string
	ValueType  string
	Name       string
}

var (
	cpuReservedCoresMD = &metricMetadata{
		MetricKind: "GAUGE",
		ValueType:  "DOUBLE",
		Name:       "container.googleapis.com/container/cpu/reserved_cores",
	}

	cpuUsageTimeMD = &metricMetadata{
		MetricKind: "CUMULATIVE",
		ValueType:  "DOUBLE",
		Name:       "container.googleapis.com/container/cpu/usage_time",
	}

	uptimeMD = &metricMetadata{
		MetricKind: "CUMULATIVE",
		ValueType:  "DOUBLE",
		Name:       "container.googleapis.com/container/uptime",
	}

	utilizationMD = &metricMetadata{
		MetricKind: "GAUGE",
		ValueType:  "DOUBLE",
		Name:       "container.googleapis.com/container/cpu/utilization",
	}

	networkRxMD = &metricMetadata{
		MetricKind: "CUMULATIVE",
		ValueType:  "INT64",
		Name:       "container.googleapis.com/container/network/received_bytes_count",
	}

	networkTxMD = &metricMetadata{
		MetricKind: "CUMULATIVE",
		ValueType:  "INT64",
		Name:       "container.googleapis.com/container/network/sent_bytes_count",
	}

	memoryLimitMD = &metricMetadata{
		MetricKind: "GAUGE",
		ValueType:  "INT64",
		Name:       "container.googleapis.com/container/memory/bytes_total",
	}

	memoryBytesUsedMD = &metricMetadata{
		MetricKind: "GAUGE",
		ValueType:  "INT64",
		Name:       "container.googleapis.com/container/memory/bytes_used",
	}

	memoryPageFaultsMD = &metricMetadata{
		MetricKind: "CUMULATIVE",
		ValueType:  "INT64",
		Name:       "container.googleapis.com/container/memory/page_fault_count",
	}

	diskBytesUsedMD = &metricMetadata{
		MetricKind: "GAUGE",
		ValueType:  "INT64",
		Name:       "container.googleapis.com/container/disk/bytes_used",
	}

	diskBytesTotalMD = &metricMetadata{
		MetricKind: "GAUGE",
		ValueType:  "INT64",
		Name:       "container.googleapis.com/container/disk/bytes_total",
	}
)

func (sink *stackdriverSink) Name() string {
	return "Stackdriver Sink"
}

func (sink *stackdriverSink) Stop() {
	// nothing needs to be done
}

func (sink *stackdriverSink) ExportData(dataBatch *core.DataBatch) {
	req := getReq()
	for _, metricSet := range dataBatch.MetricSets {
		switch metricSet.Labels["type"] {
		case core.MetricSetTypeNode, core.MetricSetTypePod, core.MetricSetTypePodContainer, core.MetricSetTypeSystemContainer:
		default:
			continue
		}

		if metricSet.Labels["type"] == core.MetricSetTypeNode {
			metricSet.Labels[core.LabelContainerName.Key] = "machine"
		}

		sink.preprocessMemoryMetrics(metricSet)

		for name, value := range metricSet.MetricValues {
			point := sink.translateMetric(dataBatch.Timestamp, metricSet.Labels, name, value, metricSet.CreateTime)

			if point != nil {
				req.TimeSeries = append(req.TimeSeries, point)
			}
			if len(req.TimeSeries) >= maxTimeseriesPerRequest {
				sink.sendRequest(req)
				req = getReq()
			}
		}

		for _, metric := range metricSet.LabeledMetrics {
			point := sink.translateLabeledMetric(dataBatch.Timestamp, metricSet.Labels, metric, metricSet.CreateTime)

			if point != nil {
				req.TimeSeries = append(req.TimeSeries, point)
			}
			if len(req.TimeSeries) >= maxTimeseriesPerRequest {
				sink.sendRequest(req)
				req = getReq()
			}
		}
	}

	if len(req.TimeSeries) > 0 {
		sink.sendRequest(req)
	}
}

func CreateStackdriverSink(uri *url.URL) (core.DataSink, error) {
	if len(uri.Scheme) > 0 {
		return nil, fmt.Errorf("Scheme should not be set for Stackdriver sink")
	}
	if len(uri.Host) > 0 {
		return nil, fmt.Errorf("Host should not be set for Stackdriver sink")
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

	sink := &stackdriverSink{
		project:           projectId,
		zone:              zone,
		stackdriverClient: stackdriverClient,
	}

	glog.Infof("Created Stackdriver sink")

	return sink, nil
}

func (sink *stackdriverSink) sendRequest(req *sd_api.CreateTimeSeriesRequest) {
	_, err := sink.stackdriverClient.Projects.TimeSeries.Create(fullProjectName(sink.project), req).Do()
	if err != nil {
		glog.Errorf("Error while sending request to Stackdriver %v", err)
	}
}

func (sink *stackdriverSink) preprocessMemoryMetrics(metricSet *core.MetricSet) {
	usage := metricSet.MetricValues[core.MetricMemoryUsage.MetricDescriptor.Name].IntValue
	workingSet := metricSet.MetricValues[core.MetricMemoryWorkingSet.MetricDescriptor.Name].IntValue
	bytesUsed := core.MetricValue{
		IntValue: usage - workingSet,
	}

	metricSet.MetricValues["memory/bytes_used"] = bytesUsed

	memoryFaults := metricSet.MetricValues[core.MetricMemoryPageFaults.MetricDescriptor.Name].IntValue
	majorMemoryFaults := metricSet.MetricValues[core.MetricMemoryMajorPageFaults.MetricDescriptor.Name].IntValue

	minorMemoryFaults := core.MetricValue{
		IntValue: memoryFaults - majorMemoryFaults,
	}
	metricSet.MetricValues["memory/minor_page_faults"] = minorMemoryFaults
}

func (sink *stackdriverSink) translateLabeledMetric(timestamp time.Time, labels map[string]string, metric core.LabeledMetric, createTime time.Time) *sd_api.TimeSeries {
	resourceLabels := sink.getResourceLabels(labels)
	switch metric.Name {
	case core.MetricFilesystemUsage.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, timestamp, metric.MetricValue.IntValue)
		ts := createTimeSeries(resourceLabels, diskBytesUsedMD, point)
		ts.Metric.Labels = map[string]string{
			"device_name": metric.Labels[core.LabelResourceID.Key],
		}
		return ts
	case core.MetricFilesystemLimit.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, timestamp, metric.MetricValue.IntValue)
		ts := createTimeSeries(resourceLabels, diskBytesTotalMD, point)
		ts.Metric.Labels = map[string]string{
			"device_name": metric.Labels[core.LabelResourceID.Key],
		}
		return ts
	default:
		return nil
	}
}

func (sink *stackdriverSink) translateMetric(timestamp time.Time, labels map[string]string, name string, value core.MetricValue, createTime time.Time) *sd_api.TimeSeries {
	resourceLabels := sink.getResourceLabels(labels)
	switch name {
	case core.MetricUptime.MetricDescriptor.Name:
		doubleValue := float64(value.IntValue) / float64(time.Second/time.Millisecond)
		point := sink.doublePoint(timestamp, createTime, doubleValue)
		return createTimeSeries(resourceLabels, uptimeMD, point)
	case core.MetricCpuLimit.MetricDescriptor.Name:
		point := sink.doublePoint(timestamp, timestamp, float64(value.FloatValue))
		return createTimeSeries(resourceLabels, cpuReservedCoresMD, point)
	case core.MetricCpuUsage.MetricDescriptor.Name:
		point := sink.doublePoint(timestamp, createTime, float64(value.FloatValue))
		return createTimeSeries(resourceLabels, cpuUsageTimeMD, point)
	case core.MetricNetworkRx.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, createTime, value.IntValue)
		return createTimeSeries(resourceLabels, networkRxMD, point)
	case core.MetricNetworkTx.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, createTime, value.IntValue)
		return createTimeSeries(resourceLabels, networkTxMD, point)
	case core.MetricMemoryLimit.MetricDescriptor.Name:
		// omit nodes, using memory/node_allocatable instead
		if labels["type"] == core.MetricSetTypeNode {
			return nil
		}
		point := sink.intPoint(timestamp, timestamp, value.IntValue)
		return createTimeSeries(resourceLabels, memoryLimitMD, point)
	case core.MetricNodeMemoryAllocatable.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, timestamp, value.IntValue)
		return createTimeSeries(resourceLabels, memoryLimitMD, point)
	case core.MetricMemoryMajorPageFaults.MetricDescriptor.Name:
		point := sink.intPoint(timestamp, createTime, value.IntValue)
		ts := createTimeSeries(resourceLabels, memoryPageFaultsMD, point)
		ts.Metric.Labels = map[string]string{
			"fault_type": "major",
		}
		return ts
	case "memory/bytes_used":
		point := sink.intPoint(timestamp, timestamp, value.IntValue)
		return createTimeSeries(resourceLabels, memoryBytesUsedMD, point)
	case "memory/minor_page_faults":
		point := sink.intPoint(timestamp, createTime, value.IntValue)
		ts := createTimeSeries(resourceLabels, memoryPageFaultsMD, point)
		ts.Metric.Labels = map[string]string{
			"fault_type": "minor",
		}
		return ts
	default:
		return nil
	}
}

func (sink *stackdriverSink) getResourceLabels(labels map[string]string) map[string]string {
	return map[string]string{
		"project_id":     sink.project,
		"cluster_name":   "",
		"zone":           sink.zone,
		"instance_id":    labels[core.LabelHostID.Key],
		"namespace_id":   labels[core.LabelPodNamespaceUID.Key],
		"pod_id":         labels[core.LabelPodId.Key],
		"container_name": labels[core.LabelContainerName.Key],
	}
}

func createTimeSeries(resourceLabels map[string]string, metadata *metricMetadata, point *sd_api.Point) *sd_api.TimeSeries {
	return &sd_api.TimeSeries{
		Metric: &sd_api.Metric{
			Type: metadata.Name,
		},
		MetricKind: metadata.MetricKind,
		ValueType:  metadata.ValueType,
		Resource: &sd_api.MonitoredResource{
			Labels: resourceLabels,
			Type:   "gke_container",
		},
		Points: []*sd_api.Point{point},
	}
}

func (sink *stackdriverSink) doublePoint(endTime time.Time, startTime time.Time, value float64) *sd_api.Point {
	return &sd_api.Point{
		Interval: &sd_api.TimeInterval{
			EndTime:   endTime.Format(time.RFC3339),
			StartTime: startTime.Format(time.RFC3339),
		},
		Value: &sd_api.TypedValue{
			DoubleValue:     value,
			ForceSendFields: []string{"DoubleValue"},
		},
	}

}

func (sink *stackdriverSink) intPoint(endTime time.Time, startTime time.Time, value int64) *sd_api.Point {
	return &sd_api.Point{
		Interval: &sd_api.TimeInterval{
			EndTime:   endTime.Format(time.RFC3339),
			StartTime: startTime.Format(time.RFC3339),
		},
		Value: &sd_api.TypedValue{
			Int64Value:      value,
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
