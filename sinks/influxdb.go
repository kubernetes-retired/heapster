package sinks

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info"
	influxdb "github.com/influxdb/influxdb/client"
	"github.com/vishh/heapster/sources"
)

var (
	argBufferDuration = flag.Duration("sink_influxdb_buffer_duration", 10*time.Second, "Time duration for which stats should be buffered in influxdb sink before being written as a single transaction")
	argDbUsername     = flag.String("sink_influxdb_username", "root", "InfluxDB username")
	argDbPassword     = flag.String("sink_influxdb_password", "root", "InfluxDB password")
	argDbHost         = flag.String("sink_influxdb_host", "localhost:8086", "InfluxDB host:port")
	argDbName         = flag.String("sink_influxdb_name", "k8s", "Influxdb database name")
)

const (
	statsTable            string = "stats"
	colTimestamp          string = "timestamp"
	colPodName            string = "pod"
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
	client         *influxdb.Client
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
	labels := []string{}
	for key, value := range pod.Labels {
		labels = append(labels, fmt.Sprintf("%s:%s", key, value))
	}
	columns = append(columns, colLabel)
	values = append(values, strings.Join(labels, ","))

	// Container name
	columns = append(columns, colContainerName)
	values = append(values, container.Name)

	// Cumulative Cpu Usage
	columns = append(columns, colCpuCumulativeUsage)
	values = append(values, stat.Cpu.Usage.Total)

	// Memory Usage
	columns = append(columns, colMemoryUsage)
	values = append(values, stat.Memory.Usage)

	// Memory Page Faults
	columns = append(columns, colMemoryPgFaults)
	values = append(values, stat.Memory.ContainerData.Pgfault)

	// Working set size
	columns = append(columns, colMemoryWorkingSet)
	values = append(values, stat.Memory.WorkingSet)

	// Optional: Network stats.
	if stat.Network != nil {
		columns = append(columns, colRxBytes)
		values = append(values, stat.Network.RxBytes)

		columns = append(columns, colRxErrors)
		values = append(values, stat.Network.RxErrors)

		columns = append(columns, colTxBytes)
		values = append(values, stat.Network.TxBytes)

		columns = append(columns, colTxErrors)
		values = append(values, stat.Network.TxErrors)
	}
	return
}

// Returns a new influxdb series.
func (self *InfluxdbSink) newSeries(tableName string, columns []string, points []interface{}) *influxdb.Series {
	out := &influxdb.Series{
		Name:    tableName,
		Columns: columns,
		// There's only one point for each stats
		Points: make([][]interface{}, 1),
	}
	out.Points[0] = points
	return out
}

func (self *InfluxdbSink) handleContainersDataSeries(pods []sources.Pod) {
	for _, pod := range pods {
		for _, container := range pod.Containers {
			for _, stat := range container.Stats {
				col, val := self.containerStatsToValues(&pod, &container, stat)
				self.series = append(self.series, self.newSeries(statsTable, col, val))
			}
		}
	}
}

func (self *InfluxdbSink) readyToFlush() bool {
	return time.Since(self.lastWrite) >= self.bufferDuration
}

func (self *InfluxdbSink) StoreData(data Data) error {
	var seriesToFlush []*influxdb.Series
	if data, ok := data.([]sources.Pod); ok {
		self.handleContainersDataSeries(data)
	}
	if self.readyToFlush() {
		seriesToFlush = self.series
		self.series = make([]*influxdb.Series, 0)
		self.lastWrite = time.Now()
	}

	if len(seriesToFlush) > 0 {
		glog.V(2).Info("flushed data to influxdb sink")
		// TODO(vishh): Do writes in a separate thread.
		err := self.client.WriteSeries(seriesToFlush)
		if err != nil {
			glog.Errorf("failed to write stats to influxDb - %s", err)
		}
	}

	return nil
}

func NewInfluxdbSink() (Sink, error) {
	config := &influxdb.ClientConfig{
		Host:     *argDbHost,
		Username: *argDbUsername,
		Password: *argDbPassword,
		Database: *argDbName,
		IsSecure: false,
	}
	client, err := influxdb.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.DisableCompression()
	if err := client.CreateDatabase(*argDbName); err != nil {
		glog.Infof("Database creation failed - %s", err)
	}
	// Create the database if it does not already exist. Ignore errors.
	return &InfluxdbSink{
		client:         client,
		series:         make([]*influxdb.Series, 0),
		dbName:         *argDbName,
		bufferDuration: *argBufferDuration,
		lastWrite:      time.Now(),
	}, nil
}
