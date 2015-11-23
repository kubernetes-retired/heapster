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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/heapster/core"
	"k8s.io/heapster/version"

	"github.com/golang/glog"
	influxdb "github.com/influxdb/influxdb/client"
)

type influxdbClient interface {
	Write(influxdb.BatchPoints) (*influxdb.Response, error)
	Query(influxdb.Query) (*influxdb.Response, error)
	Ping() (time.Duration, string, error)
}

type influxdbSink struct {
	client influxdbClient
	sync.RWMutex
	c        config
	dbExists bool
}

type config struct {
	user     string
	password string
	host     string
	dbName   string
	secure   bool
}

const (
	// Value Field name
	valueField = "value"
	// Event special tags
	dbNotFoundError = "database not found"
)

func (sink *influxdbSink) ExportData(dataBatch *core.DataBatch) {
	if err := sink.createDatabase(); err != nil {
		glog.Errorf("Failed to create infuxdb: %v", err)
	}
	// TODO: Divide into multiple batches.
	dataPoints := make([]influxdb.Point, 0, 0)
	for _, metricSet := range dataBatch.MetricSets {
		for metricName, metricValue := range metricSet.MetricValues {
			point := influxdb.Point{
				Measurement: metricName,
				Tags:        metricSet.Labels,
				Fields: map[string]interface{}{
					"value": metricValue,
				},
				Time: dataBatch.Timestamp.UTC(),
			}
			dataPoints = append(dataPoints, point)
		}
	}
	bp := influxdb.BatchPoints{
		Points:          dataPoints,
		Database:        sink.c.dbName,
		RetentionPolicy: "default",
	}

	start := time.Now()
	if _, err := sink.client.Write(bp); err != nil {
		if strings.Contains(err.Error(), dbNotFoundError) {
			sink.dbExists = false
		}
		glog.Errorf("Failed to write stats to influxDB - %v", err)
		return
	}
	end := time.Now()
	glog.V(4).Info("Exported %d data to influxDB in %s", len(dataPoints), end.Sub(start))
}

func (sink *influxdbSink) Name() string {
	return "InfluxDB Sink"
}

func (sink *influxdbSink) Stop() {
	// nothing needs to be done.
}

func (sink *influxdbSink) createDatabase() error {
	sink.Lock()
	defer sink.Unlock()
	if sink.dbExists {
		return nil
	}
	q := influxdb.Query{
		Command: fmt.Sprintf("CREATE DATABASE %s", sink.c.dbName),
	}
	if resp, err := sink.client.Query(q); err != nil {
		if !(resp != nil && resp.Err != nil && strings.Contains(resp.Err.Error(), "already exists")) {
			return fmt.Errorf("Database creation failed: %v", err)
		}
	}
	sink.dbExists = true
	glog.Infof("Created database %q on influxDB server at %q", sink.c.dbName, sink.c.host)
	return nil
}

// Returns a thread-compatible implementation of influxdb interactions.
func new(c config) (core.DataSink, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   c.host,
	}
	if c.secure {
		url.Scheme = "https"
	}

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
	if _, _, err := client.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping InfluxDB server at %q - %v", c.host, err)
	}
	return &influxdbSink{
		client: client,
		c:      c,
	}, nil
}

func CreateInfluxdbSink(uri *url.URL) (core.DataSink, error) {
	defaultConfig := config{
		user:     "root",
		password: "root",
		host:     "localhost:8086",
		dbName:   "k8s",
		secure:   false,
	}

	if len(uri.Host) > 0 {
		defaultConfig.host = uri.Host
	}
	opts := uri.Query()
	if len(opts["user"]) >= 1 {
		defaultConfig.user = opts["user"][0]
	}
	// TODO: use more secure way to pass the password.
	if len(opts["pw"]) >= 1 {
		defaultConfig.password = opts["pw"][0]
	}
	if len(opts["db"]) >= 1 {
		defaultConfig.dbName = opts["db"][0]
	}
	if len(opts["secure"]) >= 1 {
		val, err := strconv.ParseBool(opts["secure"][0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse `secure` flag - %v", err)
		}
		defaultConfig.secure = val
	}
	sink, err := new(defaultConfig)
	if err != nil {
		return nil, err
	}
	glog.Infof("created influxdb sink with options: %v", defaultConfig)

	return sink, nil
}
