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

package sinks

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sinks/gcm"
	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/golang/glog"
)

type GcmSink struct {
	// The driver for GCM interactions.
	driver *gcm.Driver

	// The metrics currently supported.
	supportedMetrics []gcm.SupportedMetric

	// TODO(vmarmol): Garbage collect these, as it stands we will always keep them around.
	// Map of timeseries to the time of the last point we exported.
	lastExported map[timeseriesKey]time.Time
}

// For GCM, a timeseries is a unique pairing of metric name and labels with their values.
type timeseriesKey struct {
	// Name of the metric.
	Name string

	// Mangled labels on the metric.
	Labels string
}

func NewGcmSink() (*GcmSink, error) {
	driver, err := gcm.NewDriver()
	if err != nil {
		return nil, err
	}

	// Get supported metrics.
	supportedMetrics := gcm.GetSupportedMetrics()
	for i := range supportedMetrics {
		supportedMetrics[i].Labels = allLabels
	}

	// Create the metrics.
	descriptors := make([]gcm.MetricDescriptor, 0, len(supportedMetrics))
	for _, supported := range supportedMetrics {
		descriptors = append(descriptors, supported.MetricDescriptor)
	}
	err = driver.AddMetrics(descriptors)
	if err != nil {
		return nil, err
	}

	return &GcmSink{
		driver:           driver,
		supportedMetrics: supportedMetrics,
		lastExported:     make(map[timeseriesKey]time.Time),
	}, nil
}

// TODO(vmarmol): Paralellize this.
func (self *GcmSink) StoreData(input Data) error {
	data, ok := input.(api.AggregateData)
	if !ok {
		return fmt.Errorf("requesting unrecognized type to be stored in GCM")
	}

	// Format metrics and push them.
	var lastErr error
	for _, pod := range data.Pods {
		err := self.pushPodMetrics(&pod)
		if err != nil {
			lastErr = err
		}
	}
	err := self.pushContainerSliceMetrics(data.Containers)
	if err != nil {
		lastErr = err
	}
	err = self.pushContainerSliceMetrics(data.Machine)
	if err != nil {
		lastErr = err
	}

	return lastErr
}

func (self *GcmSink) pushPodMetrics(pod *api.Pod) error {
	// Generate the labels.
	labels := make(map[string]string)
	labels[labelPodId] = pod.ID
	labels[labelLabels] = gcm.LabelsToString(pod.Labels)
	labels[labelHostname] = pod.Hostname

	// Break the individual metrics from the container statistics.
	var lastErr error
	for _, container := range pod.Containers {
		err := self.pushContainerMetrics(container, labels)
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (self *GcmSink) pushContainerSliceMetrics(containers []api.Container) error {
	labels := make(map[string]string)
	var lastErr error
	for _, container := range containers {
		labels[labelHostname] = container.Hostname
		err := self.pushContainerMetrics(&container, labels)
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (self *GcmSink) pushContainerMetrics(container *api.Container, labels map[string]string) error {
	labels[labelContainerName] = container.Name

	// One metric value per data point.
	var lastErr error
	labelsAsString := gcm.LabelsToString(labels)
	for _, stat := range container.Stats {
		// Add all supported metrics that have values.
		m := make([]gcm.Metric, 0, len(self.supportedMetrics))
		for _, supported := range self.supportedMetrics {
			// GCM won't accept metrics older than the last point we pushed so ignore any stats
			// older than the last push we did.
			key := timeseriesKey{
				Name:   supported.Name,
				Labels: labelsAsString,
			}
			if self.lastExported[key].After(stat.Timestamp) {
				continue
			}

			if supported.HasValue(&container.Spec) {
				// Cumulative stats have container creation time as their start time.
				var startTime time.Time
				if supported.Type == gcm.MetricCumulative {
					startTime = container.Spec.CreationTime
				} else {
					startTime = stat.Timestamp
				}

				m = append(m, gcm.Metric{
					Name:   supported.Name,
					Labels: labels,
					Start:  startTime,
					End:    stat.Timestamp,
					Value:  supported.GetValue(stat),
				})
				self.lastExported[key] = stat.Timestamp
			}
		}

		// Skip pushing if there are no metrics.
		if len(m) == 0 {
			continue
		}

		err := self.driver.PushMetrics(m)
		if err != nil {
			lastErr = err
			glog.Warningf("[GCM] Failed to push metrics for container %q with labels %q: %v", container.Name, labelsAsString, err)
		}
	}

	return lastErr
}

func (self *GcmSink) GetConfig() string {
	desc := "Sink type: GCM\n"

	// Add metrics being exported.
	desc += "\tExported metrics:"
	for _, supported := range self.supportedMetrics {
		desc += fmt.Sprintf("\t\t%s: %s", supported.Name, supported.Description)
	}

	// Add labels being used.
	desc += "\tExported labels:"
	for _, label := range allLabels {
		desc += fmt.Sprintf("\t\t%s: %s", label.Key, label.Description)
	}

	desc += "\n"
	return desc
}

const (
	labelPodId         = "pod_id"
	labelContainerName = "container_name"
	labelLabels        = "labels"
	labelHostname      = "hostname"
)

// TODO(vmarmol): Things we should consider adding (note that we only get 10 labels):
// - POD name, container name, and host IP: Useful to users but maybe we should just mangle them with ID and IP
// - Namespace: Are IDs unique only per namespace? If so, mangle it into the ID.
var allLabels = []gcm.LabelDescriptor{
	{
		Key:         labelPodId,
		Description: "The unique ID of the pod",
	},
	{
		Key:         labelContainerName,
		Description: "User-provided name of the container or full container name for system containers",
	},
	{
		Key:         labelLabels,
		Description: "Comma-separated list of user-provided labels",
	},
	{
		Key:         labelHostname,
		Description: "Hostname where the container ran",
	},
}
