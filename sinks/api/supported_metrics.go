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
	cadvisor "github.com/google/cadvisor/info/v1"
)

// Stub out for testing
var timeSince = time.Since

var statMetrics = []SupportedStatMetric{
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "uptime",
			Description: "Number of milliseconds since the container was started",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsMilliseconds,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return !spec.CreationTime.IsZero()
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: timeSince(spec.CreationTime).Nanoseconds() / time.Millisecond.Nanoseconds()}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "cpu/usage",
			Description: "Cumulative CPU usage on all cores",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsNanoseconds,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasCpu
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Cpu.Usage.Total)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "cpu/limit",
			Description: "CPU limit in millicores",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
			Units:       UnitsCount,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasCpu && (spec.Cpu.Limit > 0)
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			// Normalize to a conversion factor of 1000.
			return []internalPoint{{value: int64(spec.Cpu.Limit*1000) / 1024}}
		},
		OnlyExportIfChanged: true,
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/usage",
			Description: "Total memory usage",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasMemory
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Memory.Usage)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/working_set",
			Description: "Total working set usage. Working set is the memory being used and not easily dropped by the kernel",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasMemory
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Memory.WorkingSet)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/limit",
			Description: "Memory limit",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasMemory && (spec.Memory.Limit > 0)
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(spec.Memory.Limit)}}
		},
		OnlyExportIfChanged: true,
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/page_faults",
			Description: "Number of page faults",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsCount,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasMemory
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Memory.ContainerData.Pgfault)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/major_page_faults",
			Description: "Number of major page faults",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsCount,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasMemory
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Memory.ContainerData.Pgmajfault)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "network/rx",
			Description: "Cumulative number of bytes received over the network",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasNetwork
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Network.RxBytes)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "network/rx_errors",
			Description: "Cumulative number of errors while receiving over the network",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsCount,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasNetwork
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Network.RxErrors)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "network/tx",
			Description: "Cumulative number of bytes sent over the network",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasNetwork
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Network.TxBytes)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "network/tx_errors",
			Description: "Cumulative number of errors while sending over the network",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsCount,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasNetwork
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			return []internalPoint{{value: int64(stat.Network.TxErrors)}}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "filesystem/usage",
			Description: "Total number of bytes consumed on a filesystem",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasFilesystem
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			result := make([]internalPoint, 0, len(stat.Filesystem))
			for _, fs := range stat.Filesystem {
				result = append(result, internalPoint{
					value: int64(fs.Usage),
					labels: map[string]string{
						LabelResourceID: fs.Device,
					},
				})
			}
			return result
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "filesystem/limit",
			Description: "The total size of filesystem in bytes",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasFilesystem
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			result := make([]internalPoint, 0, len(stat.Filesystem))
			for _, fs := range stat.Filesystem {
				result = append(result, internalPoint{
					value: int64(fs.Limit),
					labels: map[string]string{
						LabelResourceID: fs.Device,
					},
				})
			}
			return result
		},
		OnlyExportIfChanged: true,
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "cpu/node_usage",
			Description: "Cumulative CPU usage on all cores on a node",
			Type:        MetricCumulative,
			ValueType:   ValueInt64,
			Units:       UnitsNanoseconds,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasCpu && spec.HasResourceId
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			result := internalPoint{
				value: int64(stat.Cpu.Usage.Total),
				labels: map[string]string{
					LabelComputeResourceID:   spec.ResourceId,
					LabelComputeResourceType: "instance",
				},
			}
			return []internalPoint{result}
		},
	},
	{
		MetricDescriptor: MetricDescriptor{
			Name:        "memory/node_usage",
			Description: "Total memory usage on a node",
			Type:        MetricGauge,
			ValueType:   ValueInt64,
			Units:       UnitsBytes,
		},
		HasValue: func(spec *source_api.ContainerSpec) bool {
			return spec.HasMemory && spec.HasResourceId
		},
		GetValue: func(spec *source_api.ContainerSpec, stat *cadvisor.ContainerStats) []internalPoint {
			result := internalPoint{
				value: int64(stat.Memory.Usage),
				labels: map[string]string{
					LabelComputeResourceID:   spec.ResourceId,
					LabelComputeResourceType: "instance",
				},
			}
			return []internalPoint{result}
		},
	},
	// TODO(vmarmol): DiskIO stats if we find those useful and know how to export them in a user-friendly way.
}

// TODO: Add Status metrics - restarts, OOMs, etc.

func SupportedStatMetrics() []SupportedStatMetric {
	return statMetrics
}
