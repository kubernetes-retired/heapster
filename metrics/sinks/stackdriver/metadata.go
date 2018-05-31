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

package stackdriver

import google_api5 "google.golang.org/genproto/googleapis/api/metric"

var (
	// Known metrics metadata

	// Container metrics

	containerUptimeMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/container/uptime",
	}

	cpuContainerCoreUsageTimeMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/container/cpu/core_usage_time",
	}

	cpuRequestedCoresMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/container/cpu/request_cores",
	}

	cpuLimitCoresMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/container/cpu/limit_cores",
	}

	memoryContainerUsedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/memory/used_bytes",
	}

	memoryRequestedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/memory/request_bytes",
	}

	memoryLimitBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/memory/limit_bytes",
	}

	ephemeralstorageContainerUsedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/ephemeral_storage/used_bytes",
	}

	ephemeralstorageRequestedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/ephemeral_storage/request_bytes",
	}

	ephemeralstorageLimitBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/ephemeral_storage/limit_bytes",
	}

	acceleratorMemoryTotalMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/accelerator/memory_total",
	}

	acceleratorMemoryUsedMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/accelerator/memory_used",
	}

	acceleratorDutyCycleMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/accelerator/duty_cycle",
	}

	acceleratorRequestedMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/accelerator/request",
	}

	restartCountMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/container/restart_count",
	}

	// Pod metrics

	volumeUsedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/pod/volume/used_bytes",
	}

	volumeTotalBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/pod/volume/total_bytes",
	}

	networkPodReceivedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/pod/network/received_bytes_count",
	}

	networkPodSentBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/pod/network/sent_bytes_count",
	}

	// Node metrics

	cpuNodeCoreUsageTimeMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/node/cpu/core_usage_time",
	}

	cpuTotalCoresMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/node/cpu/total_cores",
	}

	cpuAllocatableCoresMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/node/cpu/allocatable_cores",
	}

	memoryNodeUsedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/memory/used_bytes",
	}

	memoryTotalBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/memory/total_bytes",
	}

	memoryAllocatableBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/memory/allocatable_bytes",
	}

	ephemeralstorageNodeUsedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/ephemeral_storage/used_bytes",
	}

	ephemeralstorageTotalBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/ephemeral_storage/total_bytes",
	}

	ephemeralstorageAllocatableBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/ephemeral_storage/allocatable_bytes",
	}

	networkNodeReceivedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/network/received_bytes_count",
	}

	networkNodeSentBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node/network/sent_bytes_count",
	}

	cpuNodeDaemonCoreUsageTimeMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "kubernetes.io/node_daemon/cpu/core_usage_time",
	}

	memoryNodeDaemonUsedBytesMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "kubernetes.io/node_daemon/memory/used_bytes",
	}

	// Old resource model metrics

	legacyUptimeMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "container.googleapis.com/container/uptime",
	}

	legacyCPUReservedCoresMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "container.googleapis.com/container/cpu/reserved_cores",
	}

	legacyCPUUsageTimeMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_DOUBLE,
		Name:       "container.googleapis.com/container/cpu/usage_time",
	}

	legacyNetworkRxMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/network/received_bytes_count",
	}

	legacyNetworkTxMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/network/sent_bytes_count",
	}

	legacyMemoryLimitMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/memory/bytes_total",
	}

	legacyMemoryBytesUsedMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/memory/bytes_used",
	}

	legacyMemoryPageFaultsMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_CUMULATIVE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/memory/page_fault_count",
	}

	legacyDiskBytesUsedMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/disk/bytes_used",
	}

	legacyDiskBytesTotalMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/disk/bytes_total",
	}

	legacyAcceleratorMemoryTotalMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/accelerator/memory_total",
	}

	legacyAcceleratorMemoryUsedMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/accelerator/memory_used",
	}

	legacyAcceleratorDutyCycleMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/accelerator/duty_cycle",
	}

	legacyAcceleratorRequestMD = &metricMetadata{
		MetricKind: google_api5.MetricDescriptor_GAUGE,
		ValueType:  google_api5.MetricDescriptor_INT64,
		Name:       "container.googleapis.com/container/accelerator/request",
	}
)
