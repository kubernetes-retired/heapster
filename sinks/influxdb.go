package sinks

import (
	"flag"
	"time"

	influxdb "github.com/influxdb/influxdb/client"
	"github.com/vishh/caggregator/sources"
)

var (
	argBufferDuration = flag.Duration("sink_influxdb_buffer_duration", 1*time.Minute, "Time duration for which stats should be buffered in influxdb sink before being written as a single transaction")
	argDbUsername     = flag.String("sink_influxdb_username", "root", "InfluxDB username")
	argDbPassword     = flag.String("sink_influxdb_password", "root", "InfluxDB password")
	argDbHost         = flag.String("sink_influxdb_host", "localhost:8086", "InfluxDB host:port")
	argDbName         = flag.String("sink_influxdb_name", "k8s", "Influxdb database name")
)

const (
	colTimestamp          string = "timestamp"
	colPod                string = "pod"
	colPodStatus          string = "pod_status"
	colPodIP              string = "pod_ip"
	colLabel              string = "label"
	colHostName           string = "hostname"
	colContainerName      string = "container_name"
	colCpuCumulativeUsage string = "cpu_cumulative_usage"
	colMemoryUsage        string = "memory_usage"
	colMemoryWorkingSet   string = "memory_working_set"
	colMemoryPgFaults     string = "page_faults"
	colCpuInstantUsage    string = "cpu_instant_usage"
	colRxBytes            string = "rx_bytes"
	colRxErrors           string = "rx_errors"
	colTxBytes            string = "tx_bytes"
	colTxErrors           string = "tx_errors"
)

type InfluxdbSink struct {
	series         []*influxdb.Series
	dbName         string
	bufferDuration time.Duration
	lastWrite      time.Time
}

func (self *InfluxdbSink) containerStatsToValues(pod *sources.Pod, container *sources.Container, stat *cadvisor.ContainerStats) (columns []string, values []interface{}) {
	// Timestamp
	columns = append(columns, colTimestamp)
	values = append(values, stat.Timestamp.Format(time.RFC3339Nano))

	// Pod name
	columns = append(columns, colPodName)
	values = append(values, pod.Name)

	// Hostname
	columns = append(columns, colHostName)
	values = append(values, pod.Hostname)

	// Pod Status
	columns = append(columns, colPodStatus)
	values = append(values, pod.Status)

	// Pod IP
	columns = append(columns, colPodIP)
	values = append(values, pod.PodIP)

	// TODO(vishh): Add labels

	// Container name
	columns = append(columns, colContainerName)
	values = append(values, container.Name)

	// Cumulative Cpu Usage
	columns = append(columns, colCpuCumulativeUsage)
	values = append(values, stat.Cpu.Usage.Total)

	// Memory Usage
	columns = append(columns, colMemoryUsage)
	values = append(values, stat.Memory.Usage)

	// Working set size
	columns = append(columns, colMemoryWorkingSet)
	values = append(values, stat.Memory.WorkingSet)

	// Optional: Network stats.
	if stat.Network != nil {
		columns = append(columns, colRxBytes)
		values = append(values, stat.Network.RxBytes)

		columns = append(columns, colRxErrors)
		values = append(values, stats.Network.RxErrors)

		columns = append(columns, colTxBytes)
		values = append(values, stat.Network.TxBytes)

		columns = append(columns, colTxErrors)
		values = append(values, stat.Network.TxErrors)
	}
}

func (self *InfluxdbSink) handleContainersDataSeries(pods []sources.Pod) error {

}

func (self *InfluxdbSink) StoreData(data Data) error {
	return nil
}

func (self *InfluxdbSink) RetrieveData(from, to time.Time, data Data) error {
	// TODO(vishh): Implement this.
	return nil
}

func NewInfluxdbSink() Sink {
	return &InfluxdbSink{
		series:         make([]*influxdb.Series, 0),
		dbName:         *argDbName,
		bufferDuration: *argBufferDuration,
		lastWrite:      time.Now(),
	}
}
