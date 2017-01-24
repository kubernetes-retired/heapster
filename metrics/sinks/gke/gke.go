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

package gke

import (
	"fmt"
	"net/url"
	"time"

	gce "cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	stackdriver "google.golang.org/api/monitoring/v3"
	gce_util "k8s.io/heapster/common/gce"
	"k8s.io/heapster/metrics/core"
)

const (
	maxTimeseriesPerRequest = 20
)

type gkeSink struct {
	project           string
	zone              string
	stackdriverClient *stackdriver.Service
}

type metricMetadata struct {
	MetricKind, ValueType, Name string
}

var (
	uptimeMD = &metricMetadata{
		MetricKind: "CUMULATIVE",
		ValueType:  "DOUBLE",
		Name:       "container.googleapis.com/container/uptime",
	}
)

func (sink *gkeSink) Name() string {
	return "GKE Sink"
}

func (sink *gkeSink) Stop() {
	// nothing needs to be done
}

func (sink *gkeSink) ExportData(dataBatch *core.DataBatch) {
	req := getReq()

	for _, metricSet := range dataBatch.MetricSets {
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
	}
}

func CreateGKESink(uri *url.URL) (core.DataSink, error) {
	if len(uri.Scheme) > 0 {
		return nil, fmt.Errorf("Scheme should not be set for GKE sink")
	}
	if len(uri.Host) > 0 {
		return nil, fmt.Errorf("Host should not be set for GKE sink")
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
	stackdriverClient, err := stackdriver.New(client)
	if err != nil {
		return nil, err
	}

	sink := &gkeSink{
		project:           projectId,
		zone:              zone,
		stackdriverClient: stackdriverClient,
	}

	glog.Infof("Created GKE sink")

	return sink, nil
}

func (sink *gkeSink) sendRequest(req *stackdriver.CreateTimeSeriesRequest) {
	_, err := sink.stackdriverClient.Projects.TimeSeries.Create(fullProjectName(sink.project), req).Do()
	if err != nil {
		glog.Errorf("Error while sending request to Stackdriver %v", err)
	} else {
		glog.V(4).Infof("Successfully sent %v timeseries to Stackdriver, project %v", len(req.TimeSeries), sink.project)
	}
}

func (sink *gkeSink) translateMetric(timestamp time.Time, labels map[string]string, name string, value core.MetricValue, createTime time.Time) *stackdriver.TimeSeries {
	switch name {
	case "uptime":
		point := sink.uptimePoint(timestamp, createTime, value)
		resourceLabels := sink.getResourceLabels(labels)
		//glog.Infof("Uptime for container: %v", resourceLabels["container_name"])
		return createTimeSeries(resourceLabels, labels, uptimeMD, point)
	default:
		//		glog.Warningf("Unknown metric %v", name)
		return nil
	}
}

func (sink *gkeSink) getResourceLabels(labels map[string]string) map[string]string {
	return map[string]string{
		"project_id":     sink.project,
		"cluster_name":   "",
		"zone":           sink.zone,
		"instance_id":    labels[core.LabelHostID.Key],
		"namespace_id":   "",
		"pod_id":         labels[core.LabelPodId.Key],
		"container_name": labels[core.LabelContainerName.Key],
	}
}

func createTimeSeries(resourceLabels map[string]string, labels map[string]string, metadata *metricMetadata, point *stackdriver.Point) *stackdriver.TimeSeries {
	return &stackdriver.TimeSeries{
		Metric: &stackdriver.Metric{
			Labels: map[string]string{},
			Type:   metadata.Name,
		},
		MetricKind: metadata.MetricKind,
		ValueType:  metadata.ValueType,
		Resource: &stackdriver.MonitoredResource{
			Labels: resourceLabels,
			Type:   "gke_container",
		},
		Points: []*stackdriver.Point{point},
	}
}

func (sink *gkeSink) uptimePoint(timestamp time.Time, createTime time.Time, value core.MetricValue) *stackdriver.Point {
	return &stackdriver.Point{
		Interval: &stackdriver.TimeInterval{
			EndTime:   timestamp.Format(time.RFC3339),
			StartTime: createTime.Format(time.RFC3339),
		},
		Value: &stackdriver.TypedValue{
			DoubleValue:     float64(value.IntValue) / float64(time.Second/time.Millisecond),
			ForceSendFields: []string{"DoubleValue"},
		},
	}
}

func fullProjectName(name string) string {
	return fmt.Sprintf("projects/%s", name)
}

func getReq() *stackdriver.CreateTimeSeriesRequest {
	return &stackdriver.CreateTimeSeriesRequest{TimeSeries: make([]*stackdriver.TimeSeries, 0)}
}
