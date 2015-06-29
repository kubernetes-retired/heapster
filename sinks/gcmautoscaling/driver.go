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

package gcmautoscaling

import (
	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"

	"github.com/GoogleCloudPlatform/heapster/extpoints"
	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
	"github.com/GoogleCloudPlatform/heapster/sinks/gcm"
	"github.com/golang/glog"
)

var (
	LabelHostname = sink_api.LabelDescriptor{
		Key:         "hostname",
		Description: "Hostname where the container ran",
	}
	LabelGCEResourceID = sink_api.LabelDescriptor{
		Key:         "compute.googleapis.com/resource_id",
		Description: "Resource id for nodes specific for GCE.",
	}
	LabelGCEResourceType = sink_api.LabelDescriptor{
		Key:         "compute.googleapis.com/resource_type",
		Description: "Resource types for nodes specific for GCE.",
	}
)

var autoscalingLabels = []sink_api.LabelDescriptor{
	LabelHostname,
	LabelGCEResourceID,
	LabelGCEResourceType,
}

type utilizationMetric struct {
	name        string
	description string
}

var autoscalingMetrics = map[string]utilizationMetric{
	"cpu/usage": {
		name:        "cpu/node_utilization",
		description: "Cpu utilization as a percentage of node capacity",
	},
	"memory/usage": {
		name:        "memory/node_utilization",
		description: "Memory utilization as a percentage of memory capacity",
	},
}

type gcmAutocalingSink struct {
	core *gcm.GcmCore
	// For given hostname remember its capacity in milicores.
	cpuCapacity map[string]int64
	// Memory capacity in bytes.
	memCapacity map[string]int64
}

// Adds the specified metrics or updates them if they already exist.
func (self gcmAutocalingSink) Register(_ []sink_api.MetricDescriptor) error {
	for _, metric := range autoscalingMetrics {
		if err := self.core.Register(metric.name, metric.description, sink_api.MetricGauge.String(), sink_api.ValueDouble.String(), autoscalingLabels); err != nil {
			return err
		}
	}
	return nil
}

// Stores events into the backend.
func (self gcmAutocalingSink) StoreEvents([]kube_api.Event) error {
	// No-op, Google Cloud Monitoring doesn't store events
	return nil
}

func isNode(metric *sink_api.Point) bool {
	return metric.Labels[sink_api.LabelContainerName.Key] == "/"
}

func (self gcmAutocalingSink) updateMachineCapacity(input []sink_api.Timeseries) {
	for _, entry := range input {
		metric := entry.Point
		if !isNode(metric) {
			continue
		}

		host := metric.Labels[sink_api.LabelHostname.Key]
		value, ok := metric.Value.(int64)
		if !ok {
			continue
		}
		if metric.Name == "cpu/limit" {
			self.cpuCapacity[host] = value
		} else if metric.Name == "memory/limit" {
			self.memCapacity[host] = value
		}
	}
}

// Pushes the specified metric values in input. The metrics must already exist.
func (self gcmAutocalingSink) StoreTimeseries(input []sink_api.Timeseries) error {
	self.updateMachineCapacity(input)

	// Build a map of metrics by name.
	metrics := make(map[string][]gcm.Timeseries)
	for _, entry := range input {
		metric := entry.Point
		// We want to export only node metrics.
		if !isNode(metric) {
			continue
		}

		labels := make(map[string]string, 3)
		labels[LabelHostname.Key] = metric.Labels[sink_api.LabelHostname.Key]
		labels[LabelGCEResourceID.Key] = metric.Labels[sink_api.LabelHostID.Key]
		labels[LabelGCEResourceType.Key] = "instance"
		metric.Labels = labels

		var ts *gcm.Timeseries
		var err error
		var newVal float64
		if metric.Name == "cpu/usage" {
			ts, err = self.core.GetEquivalentRateMetric(metric)
			if err != nil {
				return err
			}
			capacity := self.cpuCapacity[labels[LabelHostname.Key]]
			if ts == nil || capacity < 1 {
				continue
			}
			newVal = *ts.Point.DoubleValue / float64(capacity)
		} else if metric.Name == "mem/usage" {
			ts, err = self.core.GetMetric(metric)
			if err != nil {
				return err
			}
			capacity := self.memCapacity[labels[LabelHostname.Key]]
			if ts == nil || capacity < 1 {
				continue
			}
			newVal = float64(*ts.Point.Int64Value) / float64(capacity)
		} else {
			continue
		}

		name := gcm.FullMetricName(autoscalingMetrics[metric.Name].name)
		ts.TimeseriesDescriptor.Metric = name
		ts.Point.DoubleValue = &newVal

		metrics[name] = append(metrics[name], *ts)
	}

	return self.core.StoreTimeseries(metrics)
}

func (self gcmAutocalingSink) DebugInfo() string {
	return "Sink Type: GCM Autoscaling"
}

func (self gcmAutocalingSink) Name() string {
	return "Google Cloud Monitoring Sink for Autoscaling"
}

func init() {
	extpoints.SinkFactories.Register(CreateGCMScalingSink, "gcmautoscaling")
}

func CreateGCMScalingSink(_ string, _ map[string][]string) ([]sink_api.ExternalSink, error) {
	core, err := gcm.NewCore()
	sink := gcmAutocalingSink{
		core:        core,
		cpuCapacity: make(map[string]int64),
		memCapacity: make(map[string]int64),
	}
	glog.Infof("created GCM Autocaling sink")
	return []sink_api.ExternalSink{sink}, err
}
