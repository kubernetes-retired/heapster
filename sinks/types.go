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
