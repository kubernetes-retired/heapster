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
	"time"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
)

type metricDecoder struct {
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

func (self *metricDecoder) Timeseries(input source_api.AggregateData) ([]Timeseries, error) {
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
func (self *metricDecoder) getPodLabels(pod *source_api.Pod) map[string]string {
	labels := make(map[string]string)
	labels[labelPodId] = pod.ID
	labels[labelPodName] = pod.Name
	labels[labelLabels] = LabelsToString(pod.Labels, ",")
	labels[labelHostname] = pod.Hostname

	return labels
}

func (self *metricDecoder) getPodMetrics(pod *source_api.Pod) []Timeseries {
	// Break the individual metrics from the container statistics.
	result := []Timeseries{}
	for index := range pod.Containers {
		timeseries := self.getContainerMetrics(&pod.Containers[index], self.getPodLabels(pod))
		result = append(result, timeseries...)
	}

	return result
}

func (self *metricDecoder) getContainerSliceMetrics(containers []source_api.Container) []Timeseries {
	labels := make(map[string]string)
	var result []Timeseries
	for index := range containers {
		labels[labelHostname] = containers[index].Hostname
		result = append(result, self.getContainerMetrics(&containers[index], copyLabels(labels))...)
	}

	return result
}

func copyLabels(labels map[string]string) map[string]string {
	c := make(map[string]string, len(labels))
	for key, val := range labels {
		c[key] = val
	}
	return c
}

func (self *metricDecoder) getContainerMetrics(container *source_api.Container, labels map[string]string) []Timeseries {
	if container == nil {
		return nil
	}
	labels[labelContainerName] = container.Name
	// One metric value per data point.
	var result []Timeseries
	labelsAsString := LabelsToString(labels, ",")
	for _, stat := range container.Stats {
		if stat == nil {
			continue
		}
		// Add all supported metrics that have values.
		for _, supported := range self.supportedStatMetrics {
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
				seriesName := supported.Name
				if supported.Units.String() != "" {
					seriesName = fmt.Sprintf("%s_%s", seriesName, supported.Units.String())
				}
				if supported.Type.String() != "" {
					seriesName = fmt.Sprintf("%s_%s", seriesName, supported.Type.String())
				}
				internalPoints := supported.GetValue(&container.Spec, stat)
				points := []Point{}
				for _, internalPoint := range internalPoints {
					labels := copyLabels(labels)
					for name, value := range internalPoint.labels {
						labels[name] = value
					}
					point := Point{
						Labels: labels,
						Start:  startTime.Round(time.Second),
						End:    stat.Timestamp,
						Value:  internalPoint.value,
					}
					points = append(points, point)

				}
				timeseries := Timeseries{
					SeriesName:    supported.Name,
					TimePrecision: Second,
					LabelKeys:     getLabelKeys(labels),
					Points:        points,
				}
				result = append(result, timeseries)
			}
			self.lastExported[key] = stat.Timestamp
		}
	}

	return result
}

func getLabelKeys(labels map[string]string) []string {
	keys := make([]string, len(labels))

	i := 0
	for key := range labels {
		keys[i] = key
		i++
	}
	return keys

}

func NewMetricDecoder() Decoder {
	// Get supported metrics.
	return &metricDecoder{
		supportedStatMetrics: statMetrics,
		lastExported:         make(map[timeseriesKey]time.Time),
	}
}
