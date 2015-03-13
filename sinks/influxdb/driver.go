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

package influxdb

import (
	"fmt"
	"strings"
	"sync"
	"time"

	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api"
	"github.com/golang/glog"
	influxdb "github.com/influxdb/influxdb/client"
)

type influxdbSink struct {
	hostname  string
	database  string
	client    *influxdb.Client
	dbName    string
	stateLock sync.RWMutex
	// TODO(rjnagal): switch to atomic if writeFailures is the only protected data.
	writeFailures int // guarded by stateLock
	avoidColumns  bool
	seqNum        metricSequenceNum
}

func (self *influxdbSink) Register(metrics []sink_api.MetricDescriptor) error {
	// Create tags once influxDB v0.9.0 is released.
	return nil
}

func (self *influxdbSink) metricToSeries(timeseries *sink_api.Timeseries) *influxdb.Series {
	columns := []string{}
	values := []interface{}{}
	// TODO: move labels to tags once v0.9.0 is released.
	seriesName := timeseries.Point.Name
	if timeseries.MetricDescriptor.Units.String() != "" {
		seriesName = fmt.Sprintf("%s_%s", seriesName, timeseries.MetricDescriptor.Units.String())
	}
	if timeseries.MetricDescriptor.Type.String() != "" {
		seriesName = fmt.Sprintf("%s_%s", seriesName, timeseries.MetricDescriptor.Type.String())
	}

	// Add the real metric value.
	columns = append(columns, "value")
	values = append(values, timeseries.Point.Value)
	// Append labels.
	if !self.avoidColumns {
		for key, value := range timeseries.Point.Labels {
			columns = append(columns, key)
			values = append(values, value)
		}
	} else {
		seriesName = strings.Replace(seriesName, "/", "_", -1)
		seriesName = fmt.Sprintf("%s_%s", sink_api.LabelsToString(timeseries.Point.Labels, "_"), seriesName)
	}
	// Add timestamp.
	columns = append(columns, "time")
	values = append(values, timeseries.Point.End.Unix())
	// Ass sequence number
	columns = append(columns, "sequence_number")
	values = append(values, self.seqNum.Get(seriesName))

	return self.newSeries(seriesName, columns, values)
}

func (self *influxdbSink) StoreTimeseries(timeseries []sink_api.Timeseries) error {
	dataPoints := []*influxdb.Series{}
	for index := range timeseries {
		dataPoints = append(dataPoints, self.metricToSeries(&timeseries[index]))
	}
	// TODO: Group all datapoints belonging to a metric into a single series.
	// TODO: Record the average time taken to flush data.
	err := self.client.WriteSeriesWithTimePrecision(dataPoints, influxdb.Second)
	if err != nil {
		glog.Errorf("failed to write stats to influxDB - %s", err)
		self.recordWriteFailure()
	}
	glog.V(1).Info("flushed stats to influxDB")
	return err
}

// Returns a new influxdb series.
func (self *influxdbSink) newSeries(seriesName string, columns []string, points []interface{}) *influxdb.Series {
	out := &influxdb.Series{
		Name:    seriesName,
		Columns: columns,
		// There's only one point for each stats
		Points: make([][]interface{}, 1),
	}
	out.Points[0] = points
	return out
}

func (self *influxdbSink) recordWriteFailure() {
	self.stateLock.Lock()
	defer self.stateLock.Unlock()
	self.writeFailures++
}

func (self *influxdbSink) getState() string {
	self.stateLock.RLock()
	defer self.stateLock.RUnlock()
	return fmt.Sprintf("\tNumber of write failures: %d\n", self.writeFailures)
}

func (self *influxdbSink) DebugInfo() string {
	desc := "Sink Type: InfluxDB\n"
	desc += fmt.Sprintf("\tclient: Host %q, Database %q\n", self.hostname, self.database)
	desc += self.getState()
	desc += "\n"
	return desc
}

func createDatabase(databaseName string, client *influxdb.Client) error {
	createDatabase := true
	if databases, err := client.GetDatabaseList(); err == nil {
		for _, database := range databases {
			if database["name"] == databaseName {
				createDatabase = false
				break
			}
		}
	}
	if createDatabase {
		if err := client.CreateDatabase(databaseName); err != nil {
			return fmt.Errorf("Database creation failed: %v", err)
		}
		glog.Infof("Created database %q on influxdb", databaseName)
	}
	return nil
}

const (
	// Attempt database creation maxRetries times before quitting.
	maxRetries = 20
	// Sleep for waitDuration between database creation retries.
	waitDuration = 30 * time.Second
)

// Returns a thread-compatible implementation of influxdb interactions.
func NewSink(hostname, username, password, databaseName string, avoidColumns bool) (sink_api.ExternalSink, error) {
	var err error
	config := &influxdb.ClientConfig{
		Host:     hostname,
		Username: username,
		Password: password,
		Database: databaseName,
		IsSecure: false,
	}
	glog.Infof("Using influxdb on host %q with database %q", hostname, databaseName)
	client, err := influxdb.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.DisableCompression()
	for i := 0; i < maxRetries; i++ {
		err = createDatabase(databaseName, client)
		if err == nil {
			break
		}
		glog.Errorf("%s. Retrying after 30 seconds", err)
		time.Sleep(waitDuration)
	}
	if err != nil {
		return nil, err
	}
	return &influxdbSink{
		hostname:     hostname,
		database:     databaseName,
		client:       client,
		dbName:       databaseName,
		avoidColumns: avoidColumns,
		seqNum:       newMetricSequenceNum(),
	}, nil
}
