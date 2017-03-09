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

const (
	CustomMetricPrefix = "custom/"
)

// Provided by Kubelet/cadvisor.
var StandardMetrics = []Metric{
	MetricUptime,
	MetricCpuUsage,
	MetricMemoryUsage,
	MetricMemoryRSS,
	MetricMemoryCache,
	MetricMemoryWorkingSet,
	MetricMemoryPageFaults,
	MetricMemoryMajorPageFaults,
	MetricNetworkRx,
	MetricNetworkRxErrors,
	MetricNetworkTx,
	MetricNetworkTxErrors}

// Metrics computed based on cluster state using Kubernetes API.
var AdditionalMetrics = []Metric{
	MetricCpuRequest,
	MetricCpuLimit,
	MetricMemoryRequest,
	MetricMemoryLimit}

// Computed based on corresponding StandardMetrics.
var RateMetrics = []Metric{
	MetricCpuUsageRate,
	MetricMemoryPageFaultsRate,
	MetricMemoryMajorPageFaultsRate,
	MetricNetworkRxRate,
	MetricNetworkRxErrorsRate,
	MetricNetworkTxRate,
	MetricNetworkTxErrorsRate}

var RateMetricsMapping = map[string]Metric{
	MetricCpuUsage.MetricDescriptor.Name:              MetricCpuUsageRate,
	MetricMemoryPageFaults.MetricDescriptor.Name:      MetricMemoryPageFaultsRate,
	MetricMemoryMajorPageFaults.MetricDescriptor.Name: MetricMemoryMajorPageFaultsRate,
	MetricNetworkRx.MetricDescriptor.Name:             MetricNetworkRxRate,
	MetricNetworkRxErrors.MetricDescriptor.Name:       MetricNetworkRxErrorsRate,
	MetricNetworkTx.MetricDescriptor.Name:             MetricNetworkTxRate,
	MetricNetworkTxErrors.MetricDescriptor.Name:       MetricNetworkTxErrorsRate}

var LabeledMetrics = []Metric{
	MetricFilesystemUsage,
	MetricFilesystemLimit,
	MetricFilesystemAvailable,
	MetricFilesystemReadsCompleted,
	MetricFilesystemReadsMerged,
	MetricFilesystemSectorsRead,
	MetricFilesystemReadTime,
	MetricFilesystemWritesCompleted,
	MetricFilesystemWritesMerged,
	MetricFilesystemSectorsWritten,
	MetricFilesystemWriteTime,
	MetricFilesystemIoInProgress,
	MetricFilesystemIoTime,
	MetricFilesystemWeightedIoTime,
}

var NodeAutoscalingMetrics = []Metric{
	MetricNodeCpuCapacity,
	MetricNodeMemoryCapacity,
	MetricNodeCpuAllocatable,
	MetricNodeMemoryAllocatable,
	MetricNodeCpuUtilization,
	MetricNodeMemoryUtilization,
	MetricNodeCpuReservation,
	MetricNodeMemoryReservation,
}

var CpuMetrics = []Metric{
	MetricCpuLimit,
	MetricCpuRequest,
	MetricCpuUsage,
	MetricCpuUsageRate,
	MetricNodeCpuAllocatable,
	MetricNodeCpuCapacity,
	MetricNodeCpuReservation,
	MetricNodeCpuUtilization,
}
var FilesystemMetrics = []Metric{
	MetricFilesystemAvailable,
	MetricFilesystemLimit,
	MetricFilesystemUsage,
}
var MemoryMetrics = []Metric{
	MetricMemoryLimit,
	MetricMemoryMajorPageFaults,
	MetricMemoryMajorPageFaultsRate,
	MetricMemoryPageFaults,
	MetricMemoryPageFaultsRate,
	MetricMemoryRequest,
	MetricMemoryUsage,
	MetricMemoryRSS,
	MetricMemoryCache,
	MetricMemoryWorkingSet,
	MetricNodeMemoryAllocatable,
	MetricNodeMemoryCapacity,
	MetricNodeMemoryUtilization,
	MetricNodeMemoryReservation,
}
var NetworkMetrics = []Metric{
	MetricNetworkRx,
	MetricNetworkRxErrors,
	MetricNetworkRxErrorsRate,
	MetricNetworkRxRate,
	MetricNetworkTx,
	MetricNetworkTxErrors,
	MetricNetworkTxErrorsRate,
	MetricNetworkTxRate,
}

type MetricFamily string

const (
	MetricFamilyCpu        MetricFamily = "cpu"
	MetricFamilyFilesystem              = "filesystem"
	MetricFamilyMemory                  = "memory"
	MetricFamilyNetwork                 = "network"
	MetricFamilyGeneral                 = "general"
)

var MetricFamilies = map[MetricFamily][]Metric{
	MetricFamilyCpu:        CpuMetrics,
	MetricFamilyFilesystem: FilesystemMetrics,
	MetricFamilyMemory:     MemoryMetrics,
	MetricFamilyNetwork:    NetworkMetrics,
}

func MetricFamilyForName(metricName string) MetricFamily {
	for family, metrics := range MetricFamilies {
		for _, metric := range metrics {
			if metricName == metric.Name {
				return family
			}
		}
	}
	return MetricFamilyGeneral
}

var AllMetrics = append(append(append(append(StandardMetrics, AdditionalMetrics...), RateMetrics...), LabeledMetrics...),
	NodeAutoscalingMetrics...)

// Definition of Standard Metrics.
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

var MetricMemoryUsage = Metric{
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

var MetricMemoryCache = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/cache",
		Description: "Cache memory",
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
			IntValue:   int64(stat.Memory.Cache)}
	},
}

var MetricMemoryRSS = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/rss",
		Description: "RSS memory",
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
			IntValue:   int64(stat.Memory.RSS)}
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

// Definition of Additional Metrics.
var MetricCpuRequest = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/request",
		Description: "CPU request (the guaranteed amount of resources) in millicores. This metric is Kubernetes specific.",
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

var MetricMemoryLimit = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/limit",
		Description: "Memory hard limit in bytes.",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
	},
}

// Definition of Rate Metrics.
var MetricCpuUsageRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/usage_rate",
		Description: "CPU usage on all cores in millicores",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
	},
}

var MetricMemoryPageFaultsRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/page_faults_rate",
		Description: "Rate of page faults in counts per second",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricMemoryMajorPageFaultsRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/major_page_faults_rate",
		Description: "Rate of major page faults in counts per second",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNetworkRxRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/rx_rate",
		Description: "Rate of bytes received over the network in bytes per second",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNetworkRxErrorsRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/rx_errors_rate",
		Description: "Rate of errors sending over the network in errors per second",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNetworkTxRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/tx_rate",
		Description: "Rate of bytes transmitted over the network in bytes per second",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNetworkTxErrorsRate = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "network/tx_errors_rate",
		Description: "Rate of errors transmitting over the network in errors per second",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeCpuCapacity = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/node_capacity",
		Description: "Cpu capacity of a node",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeMemoryCapacity = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/node_capacity",
		Description: "Memory capacity of a node",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeCpuAllocatable = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/node_allocatable",
		Description: "Cpu allocatable of a node",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeMemoryAllocatable = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/node_allocatable",
		Description: "Memory allocatable of a node",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeCpuUtilization = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/node_utilization",
		Description: "Cpu utilization as a share of node capacity",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeMemoryUtilization = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/node_utilization",
		Description: "Memory utilization as a share of memory capacity",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeCpuReservation = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "cpu/node_reservation",
		Description: "Share of cpu that is reserved on the node",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

var MetricNodeMemoryReservation = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "memory/node_reservation",
		Description: "Share of memory that is reserved on the node",
		Type:        MetricGauge,
		ValueType:   ValueFloat,
		Units:       UnitsCount,
	},
}

// Labeled metrics

var MetricFilesystemUsage = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/usage",
		Description: "Total number of bytes consumed on a filesystem",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/usage",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricGauge,
					IntValue:   int64(fs.Usage),
				},
			})
		}
		return result
	},
}

var MetricFilesystemLimit = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/limit",
		Description: "The total size of filesystem in bytes",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/limit",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricGauge,
					IntValue:   int64(fs.Limit),
				},
			})
		}
		return result
	},
}

var MetricFilesystemAvailable = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/available",
		Description: "The number of available bytes remaining in a the filesystem",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsBytes,
		Labels:      metricLabels,
	},
}

var MetricFilesystemReadsCompleted = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/reads_completed",
		Description: "The total number of reads completed",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/reads_completed",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.ReadsCompleted),
				},
			})
		}
		return result
	},
}

var MetricFilesystemReadsMerged = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/reads_merged",
		Description: "The total number of reads merged",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/reads_merged",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.ReadsMerged),
				},
			})
		}
		return result
	},
}

var MetricFilesystemSectorsRead = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/sectors_read",
		Description: "The total number of sectors read",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/sectors_read",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.SectorsRead),
				},
			})
		}
		return result
	},
}

var MetricFilesystemReadTime = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/read_time",
		Description: "The total number of milliseconds spent by all reads",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsMilliseconds,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/read_time",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.ReadTime),
				},
			})
		}
		return result
	},
}

var MetricFilesystemWritesCompleted = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/writes_completed",
		Description: "The total number of writes completed",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/writes_completed",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.WritesCompleted),
				},
			})
		}
		return result
	},
}

var MetricFilesystemWritesMerged = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/writes_merged",
		Description: "The total number of writes merged",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/writes_merged",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.WritesMerged),
				},
			})
		}
		return result
	},
}

var MetricFilesystemSectorsWritten = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/sectors_written",
		Description: "The total number of sectors written",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/sectors_written",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.SectorsWritten),
				},
			})
		}
		return result
	},
}

var MetricFilesystemWriteTime = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/write_time",
		Description: "The total number of milliseconds spent by all writes",
		Type:        MetricCumulative,
		ValueType:   ValueInt64,
		Units:       UnitsMilliseconds,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/write_time",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricCumulative,
					IntValue:   int64(fs.WriteTime),
				},
			})
		}
		return result
	},
}

var MetricFilesystemIoInProgress = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/io_in_progress",
		Description: "Number of I/Os currently in progress",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsCount,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/io_in_progress",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricGauge,
					IntValue:   int64(fs.IoInProgress),
				},
			})
		}
		return result
	},
}

var MetricFilesystemIoTime = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/io_time",
		Description: "Number of milliseconds spent doing I/Os",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsMilliseconds,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/io_time",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricGauge,
					IntValue:   int64(fs.IoTime),
				},
			})
		}
		return result
	},
}

var MetricFilesystemWeightedIoTime = Metric{
	MetricDescriptor: MetricDescriptor{
		Name:        "filesystem/weighted_io_time",
		Description: "Number of weighted milliseconds spent doing I/Os",
		Type:        MetricGauge,
		ValueType:   ValueInt64,
		Units:       UnitsMilliseconds,
		Labels:      metricLabels,
	},
	HasLabeledMetric: func(spec *cadvisor.ContainerSpec) bool {
		return spec.HasFilesystem
	},
	GetLabeledMetric: func(spec *cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) []LabeledMetric {
		result := make([]LabeledMetric, 0, len(stat.Filesystem))
		for _, fs := range stat.Filesystem {
			result = append(result, LabeledMetric{
				Name: "filesystem/weighted_io_time",
				Labels: map[string]string{
					LabelResourceID.Key: fs.Device,
				},
				MetricValue: MetricValue{
					ValueType:  ValueInt64,
					MetricType: MetricGauge,
					IntValue:   int64(fs.WeightedIoTime),
				},
			})
		}
		return result
	},
}

func IsNodeAutoscalingMetric(name string) bool {
	for _, autoscalingMetric := range NodeAutoscalingMetrics {
		if autoscalingMetric.MetricDescriptor.Name == name {
			return true
		}
	}
	return false
}

type MetricDescriptor struct {
	// The unique name of the metric.
	Name string `json:"name,omitempty"`

	// Description of the metric.
	Description string `json:"description,omitempty"`

	// Descriptor of the labels specific to this metric.
	Labels []LabelDescriptor `json:"labels,omitempty"`

	// Type and value of metric data.
	Type      MetricType `json:"type,omitempty"`
	ValueType ValueType  `json:"value_type,omitempty"`
	Units     UnitsType  `json:"units,omitempty"`
}

// Metric represents a resource usage stat metric.
type Metric struct {
	MetricDescriptor

	// Returns whether this metric is present.
	HasValue func(*cadvisor.ContainerSpec) bool

	// Returns a slice of internal point objects that contain metric values and associated labels.
	GetValue func(*cadvisor.ContainerSpec, *cadvisor.ContainerStats) MetricValue

	// Returns whether this metric is present.
	HasLabeledMetric func(*cadvisor.ContainerSpec) bool

	// Returns a slice of internal point objects that contain metric values and associated labels.
	GetLabeledMetric func(*cadvisor.ContainerSpec, *cadvisor.ContainerStats) []LabeledMetric
}
