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
}

const (
	statsTable            string = "stats"
	specTable             string = "spec"
	machineTable          string = "machine"
	colTimestamp          string = "time"
	colPodName            string = "pod"
	colPodStatus          string = "pod_status"
	colPodIP              string = "pod_ip"
	colLabels             string = "labels"
	colHostName           string = "hostname"
	colContainerName      string = "container_name"
	colCpuCumulativeUsage string = "cpu_cumulative_usage"
	colCpuInstantUsage    string = "cpu_instant_usage"
	colMemoryUsage        string = "memory_usage"
	colMemoryWorkingSet   string = "memory_working_set"
	colMemoryPgFaults     string = "page_faults"
	colRxBytes            string = "rx_bytes"
	colRxErrors           string = "rx_errors"
	colTxBytes            string = "tx_bytes"
	colTxErrors           string = "tx_errors"
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
