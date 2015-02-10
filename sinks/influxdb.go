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
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources"
	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info"
	influxdb "github.com/influxdb/influxdb/client"
)

var (
	argBufferDuration = flag.Duration("sink_influxdb_buffer_duration", 10*time.Second, "Time duration for which stats should be buffered in influxdb sink before being written as a single transaction")
	argDbUsername     = flag.String("sink_influxdb_username", "root", "InfluxDB username")
	argDbPassword     = flag.String("sink_influxdb_password", "root", "InfluxDB password")
	argDbHost         = flag.String("sink_influxdb_host", "localhost:8086", "InfluxDB host:port")
	argDbName         = flag.String("sink_influxdb_name", "k8s", "Influxdb database name")
)

type InfluxdbSink struct {
	client         *influxdb.Client
	series         []*influxdb.Series
	dbName         string
	bufferDuration time.Duration
	lastWrite      time.Time
	stateLock      sync.RWMutex
	// TODO(rjnagal): switch to atomic if writeFailures is the only protected data.
	writeFailures int // guarded by stateLock
}

func (self *InfluxdbSink) recordWriteFailure() {
	self.stateLock.Lock()
	defer self.stateLock.Unlock()

	self.writeFailures++
}

func (self *InfluxdbSink) getState() string {
	self.stateLock.RLock()
	defer self.stateLock.RUnlock()

	return fmt.Sprintf("\tNumber of write failures: %d\n", self.writeFailures)
}

func (self *InfluxdbSink) getDefaultSeriesData(pod *sources.Pod, hostname, containerName string, stat *cadvisor.ContainerStats) (columns []string, values []interface{}) {
	// Timestamp
	columns = append(columns, colTimestamp)
	values = append(values, stat.Timestamp.Unix())

	if pod != nil {
		// Pod name
		columns = append(columns, colPodName)
		values = append(values, pod.Name)

		// Pod Status
		columns = append(columns, colPodStatus)
		values = append(values, pod.Status)

		// Pod IP
		columns = append(columns, colPodIP)
		values = append(values, pod.PodIP)

		labels := []string{}
		for key, value := range pod.Labels {
			labels = append(labels, fmt.Sprintf("%s:%s", key, value))
		}
		columns = append(columns, colLabels)
		values = append(values, strings.Join(labels, ","))
	}

	// Hostname
	columns = append(columns, colHostName)
	values = append(values, hostname)

	// Container name
	columns = append(columns, colContainerName)
	values = append(values, containerName)

	return
}

func (self *InfluxdbSink) containerFsStatsToSeries(tableName, hostname, containerName string, spec cadvisor.ContainerSpec, stat *cadvisor.ContainerStats, pod *sources.Pod) (series []*influxdb.Series) {
	if len(stat.Filesystem) == 0 {
		return
	}

	for _, fsStat := range stat.Filesystem {
		columns, values := self.getDefaultSeriesData(pod, hostname, containerName, stat)

		columns = append(columns, colFsDevice)
		values = append(values, fsStat.Device)

		columns = append(columns, colFsCapacity)
		values = append(values, fsStat.Limit)

		columns = append(columns, colFsUsage)
		values = append(values, fsStat.Usage)

		columns = append(columns, colFsIoTime)
		values = append(values, fsStat.IoTime)

		columns = append(columns, colFsIoTimeWeighted)
		values = append(values, fsStat.WeightedIoTime)

		series = append(series, self.newSeries(tableName, columns, values))
	}
	return series

}

func (self *InfluxdbSink) containerStatsToValues(pod *sources.Pod, hostname, containerName string, spec cadvisor.ContainerSpec, stat *cadvisor.ContainerStats) (columns []string, values []interface{}) {
	columns, values = self.getDefaultSeriesData(pod, hostname, containerName, stat)
	if spec.HasCpu {
		// Cumulative Cpu Usage
		columns = append(columns, colCpuCumulativeUsage)
		values = append(values, stat.Cpu.Usage.Total)
	}

	if spec.HasMemory {
		// Memory Usage
		columns = append(columns, colMemoryUsage)
		values = append(values, stat.Memory.Usage)

		// Memory Page Faults
		columns = append(columns, colMemoryPgFaults)
		values = append(values, stat.Memory.ContainerData.Pgfault)

		// Working set size
		columns = append(columns, colMemoryWorkingSet)
		values = append(values, stat.Memory.WorkingSet)
	}

	// Optional: Network stats.
	if spec.HasNetwork {
		columns = append(columns, colRxBytes)
		values = append(values, stat.Network.RxBytes)

		columns = append(columns, colRxErrors)
		values = append(values, stat.Network.RxErrors)

		columns = append(columns, colTxBytes)
		values = append(values, stat.Network.TxBytes)

		columns = append(columns, colTxErrors)
		values = append(values, stat.Network.TxErrors)
	}

	// TODO(vishh): Export DiskIo stats.
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

func (self *InfluxdbSink) handlePods(pods []sources.Pod) {
	for _, pod := range pods {
		for _, container := range pod.Containers {
			for _, stat := range container.Stats {
				col, val := self.containerStatsToValues(&pod, pod.Hostname, container.Name, container.Spec, stat)
				self.series = append(self.series, self.newSeries(statsTable, col, val))
				self.series = append(self.series, self.containerFsStatsToSeries(statsTable, pod.Hostname, container.Name, container.Spec, stat, &pod)...)

			}
		}
	}
}

func (self *InfluxdbSink) handleContainers(containers []sources.RawContainer, tableName string) {
	// TODO(vishh): Export spec into a separate table and update it whenever it changes.
	for _, container := range containers {
		for _, stat := range container.Stats {
			col, val := self.containerStatsToValues(nil, container.Hostname, container.Name, container.Spec, stat)
			self.series = append(self.series, self.newSeries(tableName, col, val))
			self.series = append(self.series, self.containerFsStatsToSeries(tableName, container.Hostname, container.Name, container.Spec, stat, nil)...)
		}
	}
}

func (self *InfluxdbSink) readyToFlush() bool {
	return time.Since(self.lastWrite) >= self.bufferDuration
}

func (self *InfluxdbSink) StoreData(ip Data) error {
	var seriesToFlush []*influxdb.Series
	if data, ok := ip.(sources.ContainerData); ok {
		self.handlePods(data.Pods)
		self.handleContainers(data.Containers, statsTable)
		self.handleContainers(data.Machine, machineTable)
	} else {
		return fmt.Errorf("Requesting unrecognized type to be stored in InfluxDB")
	}
	if self.readyToFlush() {
		seriesToFlush = self.series
		self.series = make([]*influxdb.Series, 0)
		self.lastWrite = time.Now()
	}

	if len(seriesToFlush) > 0 {
		glog.V(2).Info("flushed data to influxdb sink")
		// TODO(vishh): Do writes in a separate thread.
		err := self.client.WriteSeriesWithTimePrecision(seriesToFlush, influxdb.Second)
		if err != nil {
			glog.Errorf("failed to write stats to influxDb - %s", err)
			self.recordWriteFailure()
		}
	}

	return nil
}

func (self *InfluxdbSink) GetConfig() string {
	desc := "Sink type: Influxdb\n"
	desc += fmt.Sprintf("\tclient: Host %q, Database %q\n", *argDbHost, *argDbName)
	desc += fmt.Sprintf("\tData buffering duration: %v\n", self.bufferDuration)
	desc += self.getState()
	desc += "\n"
	return desc
}

func NewInfluxdbSink() (Sink, error) {
	config := &influxdb.ClientConfig{
		Host:     *argDbHost,
		Username: *argDbUsername,
		Password: *argDbPassword,
		Database: *argDbName,
		IsSecure: false,
	}
	glog.Infof("Using influxdb on host %q with database %q", *argDbHost, *argDbName)
	client, err := influxdb.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.DisableCompression()
	createDatabase := true
	if databases, err := client.GetDatabaseList(); err == nil {
		for _, database := range databases {
			if database["name"] == *argDbName {
				createDatabase = false
				break
			}
		}
	}
	if createDatabase {
		if err := client.CreateDatabase(*argDbName); err != nil {
			glog.Errorf("Database creation failed: %v", err)
			return nil, err
		}
	}
	return &InfluxdbSink{
		client:         client,
		series:         make([]*influxdb.Series, 0),
		dbName:         *argDbName,
		bufferDuration: *argBufferDuration,
		lastWrite:      time.Now(),
	}, nil
}
