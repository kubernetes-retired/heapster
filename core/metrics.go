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

package core

import (
	"time"

	cadvisor "github.com/google/cadvisor/info/v1"
)

var StandardMetrics = []Metric{
	MetricUptime,
	MetricCpuUsage,
	MetricCpuLimit,
	MetricCpuRequest,
	MetricMemoruUsage,
	MetricMemoryWorkingSet,
	MetricMemoryLimit,
	MetricMemoryRequest,
	MetricMemoryPageFaults,
	MetricMemoryMajorPageFaults,
	MetricNetworkRx,
	MetricNetworkRxErrors,
	MetricNetworkTx,
	MetricNetworkTxErrors}

var MetricUptime = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "uptime",
		Description: "Number of milliseconds since the container was started",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsMilliseconds,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return !spec.CreationTime.IsZero()
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   time.Since(spec.CreationTime).Nanoseconds() / time.Millisecond.Nanoseconds()}
	},
}

var MetricCpuUsage = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/usage",
		Description: "Cumulative CPU usage on all cores",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsNanoseconds,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasCpu
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   int64(stat.Cpu.Usage.Total)}
	},
}

var MetricCpuUsageRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/usage_rate",
		Description: "CPU usage on all cores in millicores",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
}

var MetricCpuLimit = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/limit",
		Description: "CPU hard limit in millicores.",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasCpu && (spec.Cpu.Limit > 0)
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		// Normalize to a conversion factor of 1000.
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricGauge,
			IntValue:   int64(spec.Cpu.Limit*1000) / 1024}
	},
}

var MetricCpuRequest = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/request",
		Description: "CPU request (the guaranteed amount of resources) in millicores. This metric is Kubernetes specific.",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
}

var MetricMemoruUsage = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/usage",
		Description: "Total memory usage",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasMemory
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricGauge,
			IntValue:   int64(stat.Memory.Usage)}
	},
}

var MetricMemoryWorkingSet = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/working_set",
		Description: "Total working set usage. Working set is the memory being used and not easily dropped by the kernel",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasMemory
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricGauge,
			IntValue:   int64(stat.Memory.WorkingSet)}
	},
}

var MetricMemoryLimit = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/limit",
		Description: "Memory hard limit in bytes.",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasMemory && (spec.Memory.Limit > 0)
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricGauge,
			IntValue:   int64(spec.Memory.Limit)}
	},
}

var MetricMemoryRequest = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/request",
		Description: "Memory request (the guaranteed amount of resources) in bytes. This metric is Kubernetes specific.",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
	},
}

var MetricMemoryPageFaults = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/page_faults",
		Description: "Number of page faults",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasMemory
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   int64(stat.Memory.ContainerData.Pgfault)}
	},
}

var MetricMemoryMajorPageFaults = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/major_page_faults",
		Description: "Number of major page faults",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasMemory
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   int64(stat.Memory.ContainerData.Pgmajfault)}
	},
}

var MetricNetworkRx = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/rx",
		Description: "Cumulative number of bytes received over the network",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasNetwork
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   int64(stat.Network.RxBytes)}
	},
}

var MetricNetworkRxErrors = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/rx_errors",
		Description: "Cumulative number of errors while receiving over the network",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasNetwork
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   int64(stat.Network.RxErrors)}
	},
}

var MetricNetworkTx = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/tx",
		Description: "Cumulative number of bytes sent over the network",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasNetwork
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   int64(stat.Network.TxBytes)}
	},
}

var MetricNetworkTxErrors = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/tx_errors",
		Description: "Cumulative number of errors while sending over the network",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
	HasValue: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasNetwork
	},
	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) MetricValue {
		return MetricValue{
			ValueType:  ValueInt64,
			MetricType: MetricCumulative,
			IntValue:   int64(stat.Network.TxErrors)}
	},
}

// TODO: figure out whether we need those metrics and align our abstraction to handle it

// {
// 	MetricDescriptor: MetricDescriptor{
// 		Name:        "filesystem/usage",
// 		Description: "Total number of bytes consumed on a filesystem",
// 		Type:        MetricGauge,
// 		ValueType:   ValueInt64,
// 		Units:       UnitsBytes,
// 		Labels:      metricLabels,
// 	},
// 	HasValue: func(spec *cadvisor.ContainerSpec) bool {
// 		return spec.HasFilesystem
// 	},
// 	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []InternalPoint {
// 		result := make([]InternalPoint, 0, len(stat.Filesystem))
// 		for _, fs := range stat.Filesystem {
// 			result = append(result, InternalPoint{
// 				Value: int64(fs.Usage),
// 				Labels: map[string]string{
// 					LabelResourceID.Key: fs.Device,
// 				},
// 			})
// 		}
// 		return result
// 	},
// },
// {
// 	MetricDescriptor: MetricDescriptor{
// 		Name:        "filesystem/limit",
// 		Description: "The total size of filesystem in bytes",
// 		Type:        MetricGauge,
// 		ValueType:   ValueInt64,
// 		Units:       UnitsBytes,
// 		Labels:      metricLabels,
// 	},
// 	HasValue: func(spec *cadvisor.ContainerSpec) bool {
// 		return spec.HasFilesystem
// 	},
// 	GetValue: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []InternalPoint {
// 		result := make([]InternalPoint, 0, len(stat.Filesystem))
// 		for _, fs := range stat.Filesystem {
// 			result = append(result, InternalPoint{
// 				Value: int64(fs.Limit),
// 				Labels: map[string]string{
// 					LabelResourceID.Key: fs.Device,
// 				},
// 			})
// 		}
// 		return result
// 	},
// 	OnlyExportIfChanged: true,
// },

type MetricDescriptor struct {
	// The unique name of the metric.
	Name string

	// Description of the metric.
	Description string

	// Descriptor of the labels specific to this metric.
	Labels []LabelDescriptor

	// Type and value of metric data.
	Type      MetricType
	ValueType ValueType
	Units     UnitsType
}

// Metric represents a resource usage stat metric.
type Metric struct {
	MetricDescriptor

	// Returns whether this metric is present.
	HasValue func(*cadvisor.ContainerSpec) bool

	// Returns a slice of internal point objects that contain metric values and associated labels.
	GetValue func(*cadvisor.ContainerSpec, *cadvisor.ContainerStats) MetricValue
}
