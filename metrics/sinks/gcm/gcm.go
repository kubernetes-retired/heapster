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

package gcm

import (
	"fmt"
	"net/url"
	"time"

	"k8s.io/heapster/metrics/core"

	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gcm "google.golang.org/api/cloudmonitoring/v2beta2"
	gce "google.golang.org/cloud/compute/metadata"
)

const (
	metricDomain    = "kubernetes.io"
	customApiPrefix = "custom.cloudmonitoring.googleapis.com"
	maxNumLabels    = 10
	// The largest number of timeseries we can write to per request.
	maxTimeseriesPerRequest = 200
)

type gcmSink struct {
	project    string
	gcmService *gcm.Service
}

func (sink *gcmSink) Name() string {
	return "GCM Sink"
}

func getReq() *gcm.WriteTimeseriesRequest {
	return &gcm.WriteTimeseriesRequest{Timeseries: make([]*gcm.TimeseriesPoint, 0)}
}

func fullLabelName(name string) string {
	return fmt.Sprintf("%s/%s/label/%s", customApiPrefix, metricDomain, name)
}

func fullMetricName(name string) string {
	return fmt.Sprintf("%s/%s/%s", customApiPrefix, metricDomain, name)
}

func (sink *gcmSink) getTimeseriesPoint(timestamp time.Time, labels map[string]string, metric string, val core.MetricValue) *gcm.TimeseriesPoint {
	point := &gcm.Point{
		Start: timestamp.Format(time.RFC3339),
		End:   timestamp.Format(time.RFC3339),
	}
	switch val.ValueType {
	case core.ValueInt64:
		point.Int64Value = &val.IntValue
	case core.ValueFloat:
		v := float64(val.FloatValue)
		point.DoubleValue = &v
	default:
		glog.Errorf("Type not supported %v in %v", val.ValueType, metric)
		return nil
	}
	// For cumulative metric use the provided start time.
	if val.MetricType == core.MetricCumulative {
		point.Start = val.Start.Format(time.RFC3339)
	}

	finalLabels := make(map[string]string)
	if core.IsNodeAutoscalingMetric(metric) {
		finalLabels[fullLabelName(core.LabelHostname.Key)] = labels[core.LabelHostname.Key]
		finalLabels[fullLabelName(core.LabelGCEResourceID.Key)] = labels[core.LabelHostID.Key]
		finalLabels[fullLabelName(core.LabelGCEResourceType.Key)] = "instance"
	} else {
		supportedLables := core.GcmLabels()
		for key, value := range labels {
			if _, ok := supportedLables[key]; ok {
				finalLabels[fullLabelName(key)] = value
			}
		}
	}
	desc := &gcm.TimeseriesDescriptor{
		Project: sink.project,
		Labels:  finalLabels,
		Metric:  fullMetricName(metric),
	}

	return &gcm.TimeseriesPoint{Point: point, TimeseriesDesc: desc}
}

func (sink *gcmSink) sendRequest(req *gcm.WriteTimeseriesRequest) {
	_, err := sink.gcmService.Timeseries.Write(sink.project, req).Do()
	if err != nil {
		glog.Errorf("Error while sending request to GCM %v", err)
	} else {
		glog.V(4).Infof("Successfully sent %v timeserieses to GCM", len(req.Timeseries))
	}
}

func (sink *gcmSink) ExportData(dataBatch *core.DataBatch) {
	req := getReq()
	for _, metricSet := range dataBatch.MetricSets {
		for metric, val := range metricSet.MetricValues {
			point := sink.getTimeseriesPoint(dataBatch.Timestamp, metricSet.Labels, metric, val)
			if point != nil {
				req.Timeseries = append(req.Timeseries, point)
			}
			if len(req.Timeseries) >= maxTimeseriesPerRequest {
				sink.sendRequest(req)
				req = getReq()
			}
		}
	}
	if len(req.Timeseries) > 0 {
		sink.sendRequest(req)
	}
}

func (sink *gcmSink) Stop() {
	// nothing needs to be done.
}

// Adds the specified metrics or updates them if they already exist.
func (sink *gcmSink) register(metrics []core.Metric) error {
	for _, metric := range metrics {
		metricName := fullMetricName(metric.MetricDescriptor.Name)
		if _, err := sink.gcmService.MetricDescriptors.Delete(sink.project, metricName).Do(); err != nil {
			glog.Infof("[GCM] Deleting metric %v failed: %v", metricName, err)
		}
		labels := make([]*gcm.MetricDescriptorLabelDescriptor, 0)

		// Node autoscaling metrics have special labels.
		if core.IsNodeAutoscalingMetric(metric.MetricDescriptor.Name) {
			for _, l := range core.GcmNodeAutoscalingLabels() {
				labels = append(labels, &gcm.MetricDescriptorLabelDescriptor{
					Key:         fullLabelName(l.Key),
					Description: l.Description,
				})
			}
		} else {
			for _, l := range core.GcmLabels() {
				labels = append(labels, &gcm.MetricDescriptorLabelDescriptor{
					Key:         fullLabelName(l.Key),
					Description: l.Description,
				})
			}
		}

		t := &gcm.MetricDescriptorTypeDescriptor{
			MetricType: metric.MetricDescriptor.Type.String(),
			ValueType:  metric.MetricDescriptor.ValueType.String(),
		}
		desc := &gcm.MetricDescriptor{
			Name:           metricName,
			Project:        sink.project,
			Description:    metric.MetricDescriptor.Description,
			Labels:         labels,
			TypeDescriptor: t,
		}
		if _, err := sink.gcmService.MetricDescriptors.Create(sink.project, desc).Do(); err != nil {
			return err
		}
	}
	return nil
}

func CreateGCMSink(uri *url.URL) (core.DataSink, error) {
	if *uri != (url.URL{}) {
		return nil, fmt.Errorf("gcm sinks don't take arguments")
	}
	// Detect project ID
	projectId, err := gce.ProjectID()
	if err != nil {
		return nil, err
	}

	// Create Google Cloud Monitoring service.
	client := oauth2.NewClient(oauth2.NoContext, google.ComputeTokenSource(""))
	gcmService, err := gcm.New(client)
	if err != nil {
		return nil, err
	}

	sink := &gcmSink{project: projectId, gcmService: gcmService}
	if err := sink.register(core.AllMetrics); err != nil {
		return nil, err
	}

	glog.Infof("created GCM sink")
	return sink, nil
}
