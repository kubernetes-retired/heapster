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

	sinksV1Api "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/util"
)

// Codec represents an engine that translated from sources.api to sink.api objects.
type Decoder interface {
	// Timeseries returns the metrics found in input as a timeseries slice.
	Timeseries(input source_api.AggregateData) ([]sinksV1Api.Timeseries, error)
}

type defaultDecoder struct {
	supportedStatMetrics []sinksV1Api.SupportedStatMetric

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

func (self *defaultDecoder) Timeseries(input source_api.AggregateData) ([]sinksV1Api.Timeseries, error) {
	var result []sinksV1Api.Timeseries
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
	labels[sinksV1Api.LabelPodId.Key] = pod.ID
	labels[sinksV1Api.LabelPodNamespace.Key] = pod.Namespace
	labels[sinksV1Api.LabelPodNamespaceUID.Key] = pod.NamespaceUID
	labels[sinksV1Api.LabelPodName.Key] = pod.Name
	labels[sinksV1Api.LabelLabels.Key] = util.LabelsToString(pod.Labels, ",")
	labels[sinksV1Api.LabelHostname.Key] = pod.Hostname
	labels[sinksV1Api.LabelHostID.Key] = pod.ExternalID

	return labels
}

func (self *defaultDecoder) getPodMetrics(pod *source_api.Pod) []sinksV1Api.Timeseries {
	// Break the individual metrics from the container statistics.
	result := []sinksV1Api.Timeseries{}
	for index := range pod.Containers {
		timeseries := self.getContainerMetrics(&pod.Containers[index], self.getPodLabels(pod))
		result = append(result, timeseries...)
	}

	return result
}

func (self *defaultDecoder) getContainerSliceMetrics(containers []source_api.Container) []sinksV1Api.Timeseries {
	labels := make(map[string]string)
	var result []sinksV1Api.Timeseries
	for index := range containers {
		labels[sinksV1Api.LabelHostname.Key] = containers[index].Hostname
		labels[sinksV1Api.LabelHostID.Key] = containers[index].ExternalID
		result = append(result, self.getContainerMetrics(&containers[index], util.CopyLabels(labels))...)
	}

	return result
}

func (self *defaultDecoder) getContainerMetrics(container *source_api.Container, labels map[string]string) []sinksV1Api.Timeseries {
	if container == nil {
		return nil
	}
	labels[sinksV1Api.LabelContainerName.Key] = container.Name
	labels[sinksV1Api.LabelContainerBaseImage.Key] = container.Image

	// One metric value per data point.
	var result []sinksV1Api.Timeseries
	labelsAsString := util.LabelsToString(labels, ",")
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
				if supported.Type == sinksV1Api.MetricCumulative {
					startTime = container.Spec.CreationTime
				} else {
					startTime = stat.Timestamp
				}
				points := supported.GetValue(&container.Spec, stat)
				for _, point := range points {
					labels := util.CopyLabels(labels)
					for name, value := range point.Labels {
						labels[name] = value
					}
					timeseries := sinksV1Api.Timeseries{
						MetricDescriptor: &self.supportedStatMetrics[index].MetricDescriptor,
						Point: &sinksV1Api.Point{
							Name:   supported.Name,
							Labels: labels,
							Start:  startTime.Round(time.Second),
							End:    stat.Timestamp,
							Value:  point.Value,
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

func NewDecoder() Decoder {
	// Get supported metrics.
	return &defaultDecoder{
		supportedStatMetrics: sinksV1Api.SupportedStatMetrics(),
		lastExported:         make(map[timeseriesKey]time.Time),
	}
}
