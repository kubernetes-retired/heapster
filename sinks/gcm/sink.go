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
	"bytes"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources"
	cadvisor "github.com/google/cadvisor/info"
)

type GcmSink struct {
	// The driver for GCM interactions.
	driver *gcmDriver

	// The metrics currently supported.
	supportedMetrics []supportedMetric
}

func NewSink() (*GcmSink, error) {
	driver, err := NewDriver()
	if err != nil {
		return nil, err
	}

	// Get supported metrics.
	supportedMetrics := allMetrics[0:]
	for i := range supportedMetrics {
		supportedMetrics[i].Labels = allLabels
	}

	// Create the metrics.
	descriptors := make([]MetricDescriptor, 0, len(supportedMetrics))
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
	}, nil
}

func (self *GcmSink) StoreData(input interface{}) error {
	data, ok := input.(sources.ContainerData)
	if !ok {
		return fmt.Errorf("requesting unrecognized type to be stored in GCM")
	}

	// Get metrics from the raw data.
	metrics := make([]Metric, 0, len(data.Pods)+len(data.Containers)+len(data.Machine))
	for _, pod := range data.Pods {
		metrics = append(metrics, self.podToMetrics(&pod)...)
	}
	metrics = append(metrics, self.rawContainersToMetrics(data.Containers)...)
	metrics = append(metrics, self.rawContainersToMetrics(data.Machine)...)

	// Push the metrics step size at a time. Try to push all the metrics even if there are errors.
	var lastErr error
	step := self.driver.MaxNumPushMetrics()
	for i := 0; i < len(metrics); i += step {
		endIndex := i + step
		if endIndex > len(metrics) {
			endIndex = len(metrics)
		}

		err := self.driver.PushMetrics(metrics[i:endIndex])
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Concatenates a map of labels into a comma-separated key=value pairs.
func labelsToString(labels map[string]string) string {
	var buffer bytes.Buffer
	first := true
	for key, value := range labels {
		if !first {
			buffer.WriteString(",")
		}
		first = false
		buffer.WriteString(key)
		buffer.WriteString("=")
		buffer.WriteString(value)
	}
	return buffer.String()
}

func (self *GcmSink) podToMetrics(pod *sources.Pod) []Metric {
	metrics := make([]Metric, 0, len(pod.Containers))

	// Generate the labels.
	labels := make(map[string]string)
	labels[labelPodId] = pod.ID
	labels[labelLabels] = labelsToString(pod.Labels)
	labels[labelHostname] = pod.Hostname

	// Break the individual metrics from the container statistics.
	for _, container := range pod.Containers {
		metrics = append(metrics, self.containerToMetrics(container, labels)...)
	}
	return metrics
}

func (self *GcmSink) rawContainersToMetrics(containers []sources.RawContainer) []Metric {
	metrics := make([]Metric, 0, len(containers))

	labels := make(map[string]string)
	for _, container := range containers {
		labels[labelHostname] = container.Hostname
		metrics = append(metrics, self.containerToMetrics(&container.Container, labels)...)
	}
	return metrics
}

func (self *GcmSink) containerToMetrics(container *sources.Container, labels map[string]string) []Metric {
	labels[labelContainerName] = container.Name

	// One metric value per data point.
	metrics := make([]Metric, 0, len(container.Stats))
	for _, stat := range container.Stats {
		// Add all supported metrics that have values.
		for _, supported := range self.supportedMetrics {
			if supported.HasValue(&container.Spec) {
				// Cumulative stats have container creation time as start time.
				var startTime time.Time
				if supported.Type == MetricCumulative {
					startTime = container.Spec.CreationTime
				} else {
					startTime = stat.Timestamp
				}

				metrics = append(metrics, Metric{
					Name:   supported.Name,
					Labels: labels,
					Start:  startTime,
					End:    stat.Timestamp,
					Value:  supported.GetValue(stat),
				})
			}
		}
	}
	return metrics
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
var allLabels = []LabelDescriptor{
	LabelDescriptor{
		Key:         labelPodId,
		Description: "The unique ID of the pod",
	},
	LabelDescriptor{
		Key:         labelContainerName,
		Description: "User-provided name of the container or full container name for system containers",
	},
	LabelDescriptor{
		Key:         labelLabels,
		Description: "Comma-separated list of user-provided labels",
	},
	LabelDescriptor{
		Key:         labelHostname,
		Description: "Hostname where the container ran",
	},
}

type supportedMetric struct {
	MetricDescriptor

	// Returns whether this metric is present.
	HasValue func(*cadvisor.ContainerSpec) bool

	// Returns the desired data point for this metric from the stats.
	GetValue func(*cadvisor.ContainerStats) interface{}
}

// TODO(vmarmol): Add the rest of the metrics.
var allMetrics = []supportedMetric{
	supportedMetric{
		MetricDescriptor: MetricDescriptor{
			Name:        "cpu/usage",
			Description: "Cumulative CPU usage on all cores",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
		},
		HasValue: func(spec *cadvisor.ContainerSpec) bool {
			return spec.HasCpu
		},
		GetValue: func(stat *cadvisor.ContainerStats) interface{} {
			return int64(stat.Cpu.Usage.Total)
		},
	},
	supportedMetric{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/usage",
			Description: "Total memory usage",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
		},
		HasValue: func(spec *cadvisor.ContainerSpec) bool {
			return spec.HasMemory
		},
		GetValue: func(stat *cadvisor.ContainerStats) interface{} {
			return int64(stat.Memory.Usage)
		},
	},
	supportedMetric{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/working_set",
			Description: "Total working set usage. Working set is the memory being used and not easily dropped by the kernel",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
		},
		HasValue: func(spec *cadvisor.ContainerSpec) bool {
			return spec.HasMemory
		},
		GetValue: func(stat *cadvisor.ContainerStats) interface{} {
			return int64(stat.Memory.WorkingSet)
		},
	},
}
