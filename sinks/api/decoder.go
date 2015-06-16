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
	"time"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
)

// Codec represents an engine that translated from sources.api to sink.api objects.
type DecoderV1 interface {
	// Timeseries returns the metrics found in input as a timeseries slice.
	Timeseries(input source_api.AggregateData) ([]Timeseries, error)
}

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

func (self *defaultDecoder) Timeseries(input source_api.AggregateData) ([]Timeseries, error) {
	var result []Timeseries
	// Format metrics and push them.
	for index := range input.Pods {
		result = append(result, self.getPodMetrics(&input.Pods[index])...)
	}
	result = append(result, self.getContainerSliceMetrics(input.Containers)...)
	result = append(result, self.getContainerSliceMetrics(input.Machine)...)

	return result, nil
}

// Generate the labels.
func (self *defaultDecoder) getPodLabels(pod *source_api.Pod) map[string]string {
	labels := make(map[string]string)
	labels[LabelPodId] = pod.ID
	labels[LabelPodNamespace] = pod.Namespace
	labels[LabelPodNamespaceUID] = pod.NamespaceUID
	labels[LabelPodName] = pod.Name
	labels[LabelLabels] = LabelsToString(pod.Labels, ",")
	labels[LabelHostname] = pod.Hostname

	return labels
}

func (self *defaultDecoder) getPodMetrics(pod *source_api.Pod) []Timeseries {
	// Break the individual metrics from the container statistics.
	result := []Timeseries{}
	for index := range pod.Containers {
		timeseries := self.getContainerMetrics(&pod.Containers[index], self.getPodLabels(pod))
		result = append(result, timeseries...)
	}

	return result
}

func (self *defaultDecoder) getContainerSliceMetrics(containers []source_api.Container) []Timeseries {
	labels := make(map[string]string)
	var result []Timeseries
	for index := range containers {
		labels[LabelHostname] = containers[index].Hostname
		result = append(result, self.getContainerMetrics(&containers[index], copyLabels(labels))...)
	}

	return result
}

func (self *defaultDecoder) getContainerMetrics(container *source_api.Container, labels map[string]string) []Timeseries {
	if container == nil {
		return nil
	}
	labels[LabelContainerName] = container.Name
	// One metric value per data point.
	var result []Timeseries
	labelsAsString := LabelsToString(labels, ",")
	for _, stat := range container.Stats {
		if stat == nil {
			continue
		}
		// Add all supported metrics that have values.
		for index, supported := range self.supportedStatMetrics {
			// Finest allowed granularity is seconds.
			stat.Timestamp = stat.Timestamp.Round(time.Second)
			key := timeseriesKey{
				Name:   supported.Name,
				Labels: labelsAsString,
			}
			// TODO: remove this once the heapster source is tested to not provide duplicate stats.
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
				points := supported.GetValue(&container.Spec, stat)
				for _, point := range points {
					labels := copyLabels(labels)
					for name, value := range point.labels {
						labels[name] = value
					}
					timeseries := Timeseries{
						MetricDescriptor: &self.supportedStatMetrics[index].MetricDescriptor,
						Point: &Point{
							Name:   supported.Name,
							Labels: labels,
							Start:  startTime.Round(time.Second),
							End:    stat.Timestamp,
							Value:  point.value,
						},
					}
					result = append(result, timeseries)
				}
			}
			self.lastExported[key] = stat.Timestamp
		}
	}

	return result
}

func NewV1Decoder() DecoderV1 {
	// Get supported metrics.
	return &defaultDecoder{
		supportedStatMetrics: statMetrics,
		lastExported:         make(map[timeseriesKey]time.Time),
	}
}
