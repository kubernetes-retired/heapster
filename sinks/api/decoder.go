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

package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
)

type defaultDecoder struct {
	supportedStatMetrics []SupportedStatMetric

	// TODO: Garbage collect data.
	// TODO: Deprecate this once we the core is fixed to never export duplicate stats.
	lastExported map[timeseriesKey]time.Time
}

type timeseriesKey struct {
	// Name of the metric.
	Name string

	// Mangled labels on the metric.
	Labels string
}

// TODO(vmarmol): Paralellize this.
func (self *defaultDecoder) Metrics(input interface{}) ([]Timeseries, error) {
	data, ok := input.(source_api.AggregateData)
	if !ok {
		return nil, fmt.Errorf("requesting unrecognized type to be stored in GCM")
	}
	var result []Timeseries
	// Format metrics and push them.
	for _, pod := range data.Pods {
		result = append(result, self.getPodMetrics(&pod)...)
	}
	result = append(result, self.getContainerSliceMetrics(data.Containers)...)
	result = append(result, self.getContainerSliceMetrics(data.Machine)...)

	return result, nil
}

func (self *defaultDecoder) getPodMetrics(pod *source_api.Pod) []Timeseries {
	// Generate the labels.
	labels := make(map[string]string)
	labels[labelPodId] = pod.ID
	labels[labelLabels] = self.labelsToString(pod.Labels)
	labels[labelHostname] = pod.Hostname

	// Break the individual metrics from the container statistics.
	var result []Timeseries
	for index := range pod.Containers {
		result = append(result, self.getContainerMetrics(&pod.Containers[index], labels)...)
	}

	return result
}

func (self *defaultDecoder) getContainerSliceMetrics(containers []source_api.Container) []Timeseries {
	labels := make(map[string]string)
	var result []Timeseries
	for _, container := range containers {
		labels[labelHostname] = container.Hostname
		result = append(result, self.getContainerMetrics(&container, labels)...)
	}

	return result
}

// Concatenates a map of labels into a comma-separated key=value pairs.
func (self *defaultDecoder) labelsToString(labels map[string]string) string {
	output := make([]string, 0, len(labels))
	for key, value := range labels {
		output = append(output, fmt.Sprintf("%s=%s", key, value))
	}

	// Sort to produce a stable output.
	sort.Strings(output)
	return strings.Join(output, ",")
}

func (self *defaultDecoder) getContainerMetrics(container *source_api.Container, labels map[string]string) []Timeseries {
	if container == nil {
		return nil
	}
	labels[labelContainerName] = container.Name
	// One metric value per data point.
	var result []Timeseries
	labelsAsString := self.labelsToString(labels)
	for _, stat := range container.Stats {
		if stat == nil {
			continue
		}
		// Add all supported metrics that have values.
		for index, supported := range self.supportedStatMetrics {
			key := timeseriesKey{
				Name:   supported.Name,
				Labels: labelsAsString,
			}
			if data, ok := self.lastExported[key]; ok && data.After(stat.Timestamp) {
				continue
			}

			if supported.HasValue(&container.Spec) {
				// Cumulative stats have container creation time as their start time.
				var startTime time.Time
				if supported.Type == MetricCumulative {
					startTime = container.Spec.CreationTime
				} else {
					startTime = stat.Timestamp
				}
				result = append(result, Timeseries{
					Metric: &Metric{
						Name:   supported.Name,
						Labels: labels,
						Start:  startTime,
						End:    stat.Timestamp,
						Value:  supported.GetValue(&container.Spec, stat),
					},
					MetricDescriptor: &self.supportedStatMetrics[index].MetricDescriptor,
				})
				self.lastExported[key] = stat.Timestamp
			}
		}
	}

	return result
}

func NewDecoder() Decoder {
	// Get supported metrics.
	return &defaultDecoder{
		supportedStatMetrics: statMetrics,
		lastExported:         make(map[timeseriesKey]time.Time),
	}
}
