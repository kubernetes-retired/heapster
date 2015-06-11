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
	"fmt"
	"strings"

	bigquery "code.google.com/p/google-api-go-client/bigquery/v2"
	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info/v1"
	bigquery_client "github.com/google/cadvisor/storage/bigquery/client"
)

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

// Big query related flags defined in bigquery_client
// clientId       = flag.String("bq_id", "", "Client ID")
// clientSecret   = flag.String("bq_secret", "notasecret", "Client Secret")
// projectId      = flag.String("bq_project_id", "", "Bigquery project ID")
// serviceAccount = flag.String("bq_account", "", "Service account email")
// pemFile        = flag.String("bq_credentials_file", "", "Credential Key file (pem)")

type bigquerySink struct {
	client *bigquery_client.Client
	rows   []map[string]interface{}
}

const (
	// Bigquery schema types
	typeTimestamp string = "TIMESTAMP"
	typeString    string = "STRING"
	typeInteger   string = "INTEGER"
)

// TODO(jnagal): Infer schema through reflection. (See bigquery/client/example)
func (self *bigquerySink) GetSchema() *bigquery.TableSchema {
	fields := make([]*bigquery.TableFieldSchema, 0)

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeTimestamp,
		Name: colTimestamp,
		Mode: "REQUIRED",
	})
	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeString,
		Name: colHostName,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeString,
		Name: colPodName,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeString,
		Name: colPodStatus,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeString,
		Name: colPodIP,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeString,
		Name: colLabels,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeString,
		Name: colContainerName,
		Mode: "REQUIRED",
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colCpuCumulativeUsage,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colMemoryUsage,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colMemoryWorkingSet,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colMemoryPgFaults,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colCpuInstantUsage,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colRxBytes,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colRxErrors,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colTxBytes,
	})

	fields = append(fields, &bigquery.TableFieldSchema{
		Type: typeInteger,
		Name: colTxErrors,
	})

	return &bigquery.TableSchema{
		Fields: fields,
	}
}

func (self *bigquerySink) containerStatsToValues(
	pod *api.Pod,
	hostname,
	containerName string,
	spec api.ContainerSpec,
	stat *cadvisor.ContainerStats) (row map[string]interface{}) {
	row = make(map[string]interface{})

	// Timestamp
	row[colTimestamp] = stat.Timestamp

	// Container name
	row[colContainerName] = containerName

	// Hostname
	row[colHostName] = hostname

	if pod != nil {
		// Pod name
		row[colPodName] = pod.Name

		// Pod Status
		row[colPodStatus] = pod.Status

		// Pod IP
		row[colPodIP] = pod.PodIP

		labels := []string{}
		for key, value := range pod.Labels {
			labels = append(labels, fmt.Sprintf("%s:%s", key, value))
		}
		row[colLabels] = strings.Join(labels, ",")
	}

	if spec.HasCpu {
		// Cumulative Cpu Usage
		row[colCpuCumulativeUsage] = stat.Cpu.Usage.Total
	}

	if spec.HasMemory {
		// Memory Usage
		row[colMemoryUsage] = stat.Memory.Usage

		row[colMemoryPgFaults] = stat.Memory.ContainerData.Pgfault

		// Working set size
		row[colMemoryWorkingSet] = stat.Memory.WorkingSet
	}

	// Optional: Network stats.
	if spec.HasNetwork {
		row[colRxBytes] = stat.Network.RxBytes
		row[colRxErrors] = stat.Network.RxErrors
		row[colTxBytes] = stat.Network.TxBytes
		row[colTxErrors] = stat.Network.TxErrors
	}

	return
}

func (self *bigquerySink) handlePods(pods []api.Pod) {
	for _, pod := range pods {
		for _, container := range pod.Containers {
			for _, stat := range container.Stats {
				self.rows = append(self.rows, self.containerStatsToValues(&pod, pod.Hostname, container.Name, container.Spec, stat))
			}
		}
	}
}

func (self *bigquerySink) handleContainers(containers []api.Container) {
	for _, container := range containers {
		for _, stat := range container.Stats {
			self.rows = append(self.rows, self.containerStatsToValues(nil, container.Hostname, container.Name, container.Spec, stat))
		}
	}
}

func (self *bigquerySink) Store(ip interface{}) error {
	if data, ok := ip.(api.AggregateData); ok {
		self.handlePods(data.Pods)
		self.handleContainers(data.Containers)
		self.handleContainers(data.Machine)
	} else {
		return fmt.Errorf("Requesting unrecognized type to be stored in InfluxDB")
	}

	// TODO(vishh): Modify the big query client to take in a series of rows.
	for _, row := range self.rows {
		err := self.client.InsertRow(row)
		if err != nil {
			glog.Error(err)
		}
	}
	self.rows = self.rows[:0]
	return nil
}

func (self *bigquerySink) DebugInfo() string {
	desc := "Sink type: BigQuery\n"
	desc += "\tDataset: cadvisor\n\n"
	return desc
}

// Create a new bigquery storage driver.
func NewBigQuerySink() (ExternalSinkManager, error) {
	bqClient, err := bigquery_client.NewClient()
	if err != nil {
		return nil, err
	}
	err = bqClient.CreateDataset("cadvisor")
	if err != nil {
		return nil, err
	}

	ret := &bigquerySink{
		client: bqClient,
		rows:   make([]map[string]interface{}, 0),
	}
	schema := ret.GetSchema()
	err = bqClient.CreateTable(statsTable, schema)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
