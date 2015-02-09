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
	cadvisor "github.com/google/cadvisor/info"
)

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
