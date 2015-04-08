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

func (sink *influxdbSink) Register(metrics []sink_api.MetricDescriptor) error {
	// Create tags once influxDB v0.9.0 is released.
	return nil
}

// Returns true if this sink supports metrics
func (sink *influxdbSink) SupportsMetrics() bool {
	return true
}

// Returns true if this sink supports events
func (sink *influxdbSink) SupportsEvents() bool {
	return true
}

func (sink *influxdbSink) StoreTimeseries(timeseriesLists []*[]sink_api.Timeseries) error {
	precisions := []sink_api.TimePrecision{sink_api.Second, sink_api.Millisecond, sink_api.Microsecond}
	for _, precision := range precisions {
		err := sink.storeTimeseriesWithPrecision(timeseriesLists, precision)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sink *influxdbSink) storeTimeseriesWithPrecision(timeseriesLists []*[]sink_api.Timeseries, precision sink_api.TimePrecision) error {
	dataPoints := []*influxdb.Series{}
	for _, timeseriesList := range timeseriesLists {
		if timeseriesList != nil {
			for _, timeseries := range *timeseriesList {
				if timeseries.TimePrecision == precision {
					if !sink.avoidColumns {
						dataPoints = append(dataPoints, sink.timeSeriesToInfluxdbSeries(&timeseries))
					} else {
						for _, point := range timeseries.Points {
							dataPoints = append(dataPoints, sink.pointToInfluxdbSeriesNoColumn(timeseries.SeriesName, point))
						}
					}
				}
			}
		}
	}

	// TODO: Group all datapoints belonging to a metric into a single series.
	// TODO: Record the average time taken to flush data.
	if len(dataPoints) > 0 {
		err := sink.client.WriteSeriesWithTimePrecision(dataPoints, getInfluxdbPrecision(precision))
		if err != nil {
			glog.Errorf("failed to write time series to influxDB - %s", err)
			for _, dataPoint := range dataPoints {
				glog.Errorf("  detail: failed to write time series (%s) to influxDB with columns (%s)", dataPoint.Name, dataPoint.Columns)
			}
			sink.recordWriteFailure()
			return err
		}
		glog.V(1).Info("Succesfully wrote %q timeseries to influxDB", precision)
	}
	return nil
}

func getInfluxdbPrecision(precision sink_api.TimePrecision) influxdb.TimePrecision {
	switch precision {
	case sink_api.Second:
		return influxdb.Second
	case sink_api.Millisecond:
		return influxdb.Millisecond
	case sink_api.Microsecond:
		return influxdb.Microsecond
	}
	return influxdb.Second
}

func (sink *influxdbSink) timeSeriesToInfluxdbSeries(timeseries *sink_api.Timeseries) *influxdb.Series {
	seriesName := timeseries.SeriesName

	columns := []string{}
	columns = append(columns, "time")            // Column 0 header
	columns = append(columns, "value")           // Column 1 header
	columns = append(columns, "sequence_number") // Column 2 header
	for _, extraColumnName := range timeseries.LabelKeys {
		columns = append(columns, extraColumnName) // Column 3+ header
	}

	points := make([][]interface{}, len(timeseries.Points))
	for i, point := range timeseries.Points {
		points[i] = make([]interface{}, len(columns))
		points[i][0] = point.End.Unix()            // Column 0 Value
		points[i][1] = point.Value                 // Column 1 Value
		points[i][2] = sink.seqNum.Get(seriesName) // Column 2 Value
		for j := 0; j < len(timeseries.LabelKeys); j++ {
			points[i][j+3] = point.Labels[timeseries.LabelKeys[j]] // Column 3+ value
		}
	}

	return &influxdb.Series{
		Name:    seriesName,
		Columns: columns,
		Points:  points,
	}
}

func (sink *influxdbSink) pointToInfluxdbSeriesNoColumn(seriesName string, point sink_api.Point) *influxdb.Series {
	// Append labels to seriesName instead of adding extra columns
	seriesName = strings.Replace(seriesName, "/", "_", -1)
	seriesName = fmt.Sprintf("%s_%s", sink_api.LabelsToString(point.Labels, "_"), seriesName)

	columns := []string{}
	columns = append(columns, "time")            // Column 0 header
	columns = append(columns, "value")           // Column 1 header
	columns = append(columns, "sequence_number") // Column 2 header

	// There's only one point per series for no columns
	points := make([][]interface{}, 1)
	points[0] = make([]interface{}, len(columns))
	points[0][0] = point.End.Unix()            // Column 0 Value
	points[0][1] = point.Value                 // Column 1 Value
	points[0][2] = sink.seqNum.Get(seriesName) // Column 2 Value

	return &influxdb.Series{
		Name:    seriesName,
		Columns: columns,
		Points:  points,
	}
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
