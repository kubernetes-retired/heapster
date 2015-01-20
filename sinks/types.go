// Copyright 2014 Google Inc. All Rights Reserved.
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

package sinks

import (
	"flag"
	"fmt"
)

var argSink = flag.String("sink", "memory", "Backend storage. Options are [memory | influxdb | bigquery]")

type Data interface{}

type Sink interface {
	StoreData(data Data) error
	GetConfig() string
}

const (
	statsTable            = "stats"
	specTable             = "spec"
	machineTable          = "machine"
	colTimestamp          = "time"
	colPodName            = "pod"
	colPodStatus          = "pod_status"
	colPodIP              = "pod_ip"
	colLabels             = "labels"
	colHostName           = "hostname"
	colContainerName      = "container_name"
	colCpuCumulativeUsage = "cpu_cumulative_usage"
	colCpuInstantUsage    = "cpu_instant_usage"
	colMemoryUsage        = "memory_usage"
	colMemoryWorkingSet   = "memory_working_set"
	colMemoryPgFaults     = "page_faults"
	colRxBytes            = "rx_bytes"
	colRxErrors           = "rx_errors"
	colTxBytes            = "tx_bytes"
	colTxErrors           = "tx_errors"
	colDiskIoServiceBytes = "diskio_service_bytes"
	colDiskIoServiced     = "diskio_serviced"
	colDiskIoQueued       = "diskio_queued"
	colDiskIoSectors      = "diskio_sectors"
	colDiskIoServiceTime  = "diskio_service_time"
	colDiskIoWaitTime     = "diskio_wait_time"
	colDiskIoMerged       = "diskio_merged"
	colDiskIoTime         = "diskio_time"
	colFsDevice           = "fs_device"
	colFsCapacity         = "fs_capacity"
	colFsUsage            = "fs_usage"
	colFsIoTime           = "fs_iotime"
	colFsIoTimeWeighted   = "fs_iotime_weighted"
)

func NewSink() (Sink, error) {
	switch *argSink {
	case "memory":
		return NewMemorySink(), nil
	case "influxdb":
		return NewInfluxdbSink()
	case "bigquery":
		return NewBigQuerySink()
	default:
		return nil, fmt.Errorf("Invalid sink specified - %s", *argSink)
	}
}
