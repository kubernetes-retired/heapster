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
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	influxdb "github.com/influxdb/influxdb/client"
	"k8s.io/heapster/extpoints"
	sink_api "k8s.io/heapster/sinks/api"
	"k8s.io/heapster/util"
	"k8s.io/heapster/version"
	kube_api "k8s.io/kubernetes/pkg/api"
)

type influxdbSink struct {
	client    *influxdb.Client
	stateLock sync.RWMutex
	// TODO(rjnagal): switch to atomic if writeFailures is the only protected data.
	writeFailures int // guarded by stateLock
	seqNum        metricSequenceNum
	c             config
}

type config struct {
	user         string
	password     string
	host         string
	dbName       string
	avoidColumns bool
}

const (
	eventsSeriesName = "log/events"
	// Attempt database creation maxRetries times before quitting.
	maxRetries = 20
	// Sleep for waitDuration between database creation retries.
	waitDuration = 30 * time.Second
)

func (sink *influxdbSink) Register(metrics []sink_api.MetricDescriptor) error {
	// Create tags once influxDB v0.9.0 is released.
	return nil
}

func (sink *influxdbSink) Unregister(metrics []sink_api.MetricDescriptor) error {
	// Like Register
	return nil
}

func (sink *influxdbSink) metricToPoint(timeseries *sink_api.Timeseries) *influxdb.Point {
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
	if !sink.c.avoidColumns {
		for key, value := range timeseries.Point.Labels {
			columns = append(columns, key)
			values = append(values, value)
		}
	} else {
		seriesName = strings.Replace(seriesName, "/", "_", -1)
		seriesName = fmt.Sprintf("%s_%s", util.LabelsToString(timeseries.Point.Labels, "_"), seriesName)
	}
	// Add timestamp.
	columns = append(columns, "time")
	values = append(values, timeseries.Point.End.Unix())
	// Ass sequence number
	columns = append(columns, "sequence_number")
	values = append(values, sink.seqNum.Get(seriesName))

	return sink.newSeries(seriesName, columns, values)
}

var eventColumns = []string{
	"time",                     // Column 0
	"sequence_number",          // Column 1
	sink_api.LabelPodId.Key,    // Column 2
	sink_api.LabelPodName.Key,  // Column 3
	sink_api.LabelHostname.Key, // Column 4
	"value",                    // Column 5
}

// Stores events into the backend.
func (sink *influxdbSink) StoreEvents(events []kube_api.Event) error {
	points := []influxdb.Point{}
	if events == nil || len(events) <= 0 {
		return nil
	}
	if !sink.c.avoidColumns {
		dataPoint, err := sink.storeEventsColumns(events)
		if err != nil {
			glog.Errorf("failed to parse events: %v", err)
			return err
		}
		points = append(points, *dataPoint)
	} else {
		for _, event := range events {
			dataPoint, err := sink.storeEventNoColumns(event)
			if err != nil {
				glog.Errorf("failed to parse events: %v", err)
				return err
			}
			points = append(points, *dataPoint)
		}
	}
	bp := influxdb.BatchPoints{
		Points:   points,
		Database: sink.c.dbName,
		Time:     time.Now(),
	}
	_, err := sink.client.Write(bp)
	if err != nil {
		glog.Errorf("failed to write events to influxDB - %s", err)
		sink.recordWriteFailure()
	} else {
		glog.V(1).Info("Successfully flushed events to influxDB")
	}
	return err

}

func (sink *influxdbSink) storeEventsColumns(events []kube_api.Event) (*influxdb.Point, error) {
	if events == nil || len(events) <= 0 {
		return nil, nil
	}
	points := make([][]interface{}, len(events))
	for i, event := range events {
		points[i] = make([]interface{}, len(eventColumns))
		points[i][0] = event.LastTimestamp.Time.UTC().Round(time.Millisecond).Unix() // Column 0 - time
		points[i][1] = hashUID(string(event.UID))                                    // Column 1 - sequence_number
		if event.InvolvedObject.Kind == "Pod" {
			points[i][2] = event.InvolvedObject.UID  // Column 2 - pod_id
			points[i][3] = event.InvolvedObject.Name // Column 3 - pod_name
		} else {
			points[i][2] = "" // Column 2 - pod_id
			points[i][3] = "" // Column 3 - pod_name
		}
		value, err := getEventValue(&event)
		if err != nil {
			return nil, err
		}
		points[i][4] = event.Source.Host // Column 4 - hostname
		points[i][5] = value             // Column 5 - value
	}
	return &influxdb.Series{
		Name:    eventsSeriesName,
		Columns: eventColumns,
		Points:  points,
	}, nil
}

func hashUID(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func (sink *influxdbSink) storeEventNoColumns(event kube_api.Event) (*influxdb.Point, error) {
	// Append labels to seriesName instead of adding extra columns
	seriesName := strings.Replace(eventsSeriesName, "/", "_", -1)
	labels := make(map[string]string)
	if event.InvolvedObject.Kind == "Pod" {
		labels[sink_api.LabelPodId.Key] = string(event.InvolvedObject.UID)
		labels[sink_api.LabelPodName.Key] = event.InvolvedObject.Name
	}
	labels[sink_api.LabelHostname.Key] = event.Source.Host
	seriesName = fmt.Sprintf("%s_%s", util.LabelsToString(labels, "_"), seriesName)

	columns := []string{}
	columns = append(columns, "time")            // Column 0
	columns = append(columns, "value")           // Column 1
	columns = append(columns, "sequence_number") // Column 2

	value, err := getEventValue(&event)
	if err != nil {
		return nil, err
	}

	// There's only one point per series for no columns
	points := make([][]interface{}, 1)
	points[0] = make([]interface{}, len(columns))
	points[0][0] = event.LastTimestamp.Time.Round(time.Millisecond).Unix() // Column 0 - time
	points[0][1] = sink.seqNum.Get(eventsSeriesName)                       // Column 1 - sequence_number
	points[0][2] = value                                                   // Column 2 - value
	return &influxdb.Series{
		Name:    seriesName,
		Columns: eventColumns,
		Points:  points,
	}, nil

}

func (sink *influxdbSink) StoreTimeseries(timeseries []sink_api.Timeseries) error {
	dataPoints := []influxdb.Point{}
	for index := range timeseries {
		dataPoints = append(dataPoints, sink.metricToSeries(&timeseries[index]))
	}
	// TODO: Group all datapoints belonging to a metric into a single series.
	// TODO: Record the average time taken to flush data.
	err := sink.client.WriteSeriesWithTimePrecision(dataPoints, influxdb.Second)
	if err != nil {
		glog.Errorf("failed to write stats to influxDB - %s", err)
		sink.recordWriteFailure()
	}
	glog.V(1).Info("flushed stats to influxDB")
	return err
}

// Generate point value for event
func getEventValue(event *kube_api.Event) (string, error) {
	bytes, err := json.MarshalIndent(event, "", " ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Returns a new influxdb series.
func (sink *influxdbSink) newSeries(seriesName string, columns []string, points []interface{}) *influxdb.Series {
	out := &influxdb.Series{
		Name:    seriesName,
		Columns: columns,
		// There's only one point for each stats
		Points: make([][]interface{}, 1),
	}
	out.Points[0] = points
	return out
}

func (sink *influxdbSink) recordWriteFailure() {
	sink.stateLock.Lock()
	defer sink.stateLock.Unlock()
	sink.writeFailures++
}

func (sink *influxdbSink) getState() string {
	sink.stateLock.RLock()
	defer sink.stateLock.RUnlock()
	return fmt.Sprintf("\tNumber of write failures: %d\n", sink.writeFailures)
}

func (sink *influxdbSink) DebugInfo() string {
	desc := "Sink Type: InfluxDB\n"
	desc += fmt.Sprintf("\tclient: Host %q, Database %q\n", sink.c.host, sink.c.dbName)
	desc += sink.getState()
	desc += "\n"
	return desc
}

func (sink *influxdbSink) Name() string {
	return "InfluxDB Sink"
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

// Returns a thread-compatible implementation of influxdb interactions.
func new(c config) (sink_api.ExternalSink, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   c.host,
	}
	// if isSecure {
	// 	url.Scheme = "https"
	// }

	iConfig := &influxdb.Config{
		URL:       *url,
		Username:  c.user,
		Password:  c.password,
		UserAgent: fmt.Sprintf("%v/%v", "heapster", version.HeapsterVersion),
	}
	client, err := influxdb.NewClient(*iConfig)

	if err != nil {
		return nil, err
	}
	for i := 0; i < maxRetries; i++ {
		err = createDatabase(c.dbName, client)
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
		client:   client,
		seqNum:   newMetricSequenceNum(),
		c:        c,
	}, nil
}

func init() {
	extpoints.SinkFactories.Register(CreateInfluxdbSink, "influxdb")
}

func CreateInfluxdbSink(uri *url.URL, _ extpoints.HeapsterConf) ([]sink_api.ExternalSink, error) {
	defaultConfig := config{
		user:         "root",
		password:     "root",
		host:         "localhost:8086",
		dbName:       "k8s",
		avoidColumns: false,
	}

	if len(uri.Host) > 0 {
		defaultConfig.host = uri.Host
	}
	opts := uri.Query()
	if len(opts["user"]) >= 1 {
		defaultConfig.user = opts["user"][0]
	}
	if len(opts["pw"]) >= 1 {
		defaultConfig.password = opts["pw"][0]
	}
	if len(opts["db"]) >= 1 {
		defaultConfig.dbName = opts["db"][0]
	}
	if len(opts["avoidColumns"]) >= 1 {
		val, err := strconv.ParseBool(opts["avoidColumns"][0])
		if err != nil {
			return nil, fmt.Errorf("invalid value %q for option 'avoidColumns' passed to influxdb sink", opts["avoidColumns"][0])
		}
		defaultConfig.avoidColumns = val
	}
	sink, err := new(defaultConfig)
	if err != nil {
		return nil, err
	}
	glog.Infof("created influxdb sink with options: %v", defaultConfig)

	return []sink_api.ExternalSink{sink}, nil
}
