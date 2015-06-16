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

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
)

type DecoderV2 interface {
	// Timeseries returns the metrics found in input as a timeseries slice.
	TimeseriesFromPods([]*cache.PodElement) ([]Timeseries, error)
	TimeseriesFromContainers([]*cache.ContainerElement) ([]Timeseries, error)
}

type v2Decoder struct {
	supportedStatMetrics []SupportedStatMetric

	// TODO: Garbage collect data.
	// TODO: Deprecate this once we the core is fixed to never export duplicate stats.
	lastExported map[timeseriesKey]time.Time
}

func (self *v2Decoder) TimeseriesFromPods(pods []*cache.PodElement) ([]Timeseries, error) {
	var result []Timeseries
	// Format metrics and push them.
	for index := range pods {
		result = append(result, self.getPodMetrics(pods[index])...)
	}
	return result, nil
}
func (self *v2Decoder) TimeseriesFromContainers(containers []*cache.ContainerElement) ([]Timeseries, error) {
	labels := make(map[string]string)
	var result []Timeseries
	for index := range containers {
		labels[LabelHostname] = containers[index].Hostname
		result = append(result, self.getContainerMetrics(containers[index], copyLabels(labels))...)
	}
	return result, nil
}

// Generate the labels.
func (self *v2Decoder) getPodLabels(pod *cache.PodElement) map[string]string {
	labels := make(map[string]string)
	labels[LabelPodId] = pod.UID
	labels[LabelPodNamespace] = pod.Namespace
	labels[LabelPodNamespaceUID] = pod.NamespaceUID
	labels[LabelPodName] = pod.Name
	labels[LabelLabels] = LabelsToString(pod.Labels, ",")
	labels[LabelHostname] = pod.Hostname
	labels[LabelExternalID] = pod.ExternalID

	return labels
}

func (self *v2Decoder) getPodMetrics(pod *cache.PodElement) []Timeseries {
	// Break the individual metrics from the container statistics.
	result := []Timeseries{}
	if pod == nil || pod.Containers == nil {
		return result
	}
	for index := range pod.Containers {
		timeseries := self.getContainerMetrics(pod.Containers[index], self.getPodLabels(pod))
		result = append(result, timeseries...)
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

func (self *v2Decoder) getContainerMetrics(container *cache.ContainerElement, labels map[string]string) []Timeseries {
	if container == nil {
		return nil
	}
	labels[LabelContainerName] = container.Name
	labels[LabelExternalID] = container.ExternalID
	// One metric value per data point.
	var result []Timeseries
	labelsAsString := LabelsToString(labels, ",")
	for _, metric := range container.Metrics {
		if metric == nil || metric.Spec == nil || metric.Stats == nil {
			continue
		}
		// Add all supported metrics that have values.
		for index, supported := range self.supportedStatMetrics {
			// Finest allowed granularity is seconds.
			metric.Stats.Timestamp = metric.Stats.Timestamp.Round(time.Second)
			key := timeseriesKey{
				Name:   supported.Name,
				Labels: labelsAsString,
			}
			// TODO: remove this once the heapster source is tested to not provide duplicate metric.Statss.

			if data, ok := self.lastExported[key]; ok && data.After(metric.Stats.Timestamp) {
				continue
			}

			if supported.HasValue(metric.Spec) {
				// Cumulative metric.Statss have container creation time as their start time.
				var startTime time.Time
				if supported.Type == MetricCumulative {
					startTime = metric.Spec.CreationTime
				} else {
					startTime = metric.Stats.Timestamp
				}
				points := supported.GetValue(metric.Spec, metric.Stats)
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
							End:    metric.Stats.Timestamp,
							Value:  point.value,
						},
					}
					result = append(result, timeseries)
				}
			}
			self.lastExported[key] = metric.Stats.Timestamp
		}

	}

	return result
}

func NewV2Decoder() DecoderV2 {
	// Get supported metrics.
	return &v2Decoder{
		supportedStatMetrics: statMetrics,
		lastExported:         make(map[timeseriesKey]time.Time),
	}
}
