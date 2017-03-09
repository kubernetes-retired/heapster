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

package riemann

import (
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/riemann/riemann-go-client"
	"k8s.io/heapster/metrics/core"
)

// Used to store the Riemann configuration specified in the Heapster cli
type riemannConfig struct {
	host      string
	ttl       float32
	state     string
	tags      []string
	batchSize int
}

// contains the riemann client, the riemann configuration, and a RWMutex
type riemannSink struct {
	client riemanngo.Client
	config riemannConfig
	sync.RWMutex
}

// creates a Riemann sink. Returns a riemannSink
func CreateRiemannSink(uri *url.URL) (core.DataSink, error) {
	// Default configuration
	c := riemannConfig{
		host:      "riemann-heapster:5555",
		ttl:       60.0,
		state:     "",
		tags:      make([]string, 0),
		batchSize: 1000,
	}
	// check host
	if len(uri.Host) > 0 {
		c.host = uri.Host
	}
	options := uri.Query()
	// check ttl
	if len(options["ttl"]) > 0 {
		var ttl, err = strconv.ParseFloat(options["ttl"][0], 32)
		if err != nil {
			return nil, err
		}
		c.ttl = float32(ttl)
	}
	// check batch size
	if len(options["batchsize"]) > 0 {
		var batchSize, err = strconv.Atoi(options["batchsize"][0])
		if err != nil {
			return nil, err
		}
		c.batchSize = batchSize
	}
	// check state
	if len(options["state"]) > 0 {
		c.state = options["state"][0]
	}
	// check tags
	if len(options["tags"]) > 0 {
		c.tags = options["tags"]
	} else {
		c.tags = []string{"heapster"}
	}

	glog.Infof("Riemann sink URI: '%+v', host: '%+v', options: '%+v', ", uri, c.host, options)
	rs := &riemannSink{
		client: nil,
		config: c,
	}
	// connect the client
	err := rs.connectRiemannClient()
	if err != nil {
		glog.Warningf("Riemann sink not connected: %v", err)
		// Warn but return the sink => the client in the sink can be nil
	}
	return rs, nil
}

// Receives a sink, connect the riemann client.
func (rs *riemannSink) connectRiemannClient() error {
	glog.Infof("Connect Riemann client...")
	client := riemanngo.NewTcpClient(rs.config.host)
	runtime.SetFinalizer(client, func(c riemanngo.Client) { c.Close() })
	// 5 seconds timeout
	err := client.Connect(5)
	if err != nil {
		return err
	}
	rs.client = client
	return nil
}

// Return a user-friendly string describing the sink
func (sink *riemannSink) Name() string {
	return "Riemann Sink"
}

func (sink *riemannSink) Stop() {
	// nothing needs to be done.
}

// Receives a list of riemanngo.Event, the sink, and parameters.
// Creates a new event using the parameters and the sink config, and add it into the Event list.
// Can send events if events is full
// Return the list.
func appendEvent(events []riemanngo.Event, sink *riemannSink, host, name string, value interface{}, labels map[string]string, timestamp int64) []riemanngo.Event {
	event := riemanngo.Event{
		Time:        timestamp,
		Service:     name,
		Host:        host,
		Description: "",
		Attributes:  labels,
		Metric:      value,
		Ttl:         sink.config.ttl,
		State:       sink.config.state,
		Tags:        sink.config.tags,
	}
	// state everywhere
	events = append(events, event)
	if len(events) >= sink.config.batchSize {
		sink.sendData(events)
		events = nil
	}
	return events
}

// ExportData Send a collection of Timeseries to Riemann
func (sink *riemannSink) ExportData(dataBatch *core.DataBatch) {
	sink.Lock()
	defer sink.Unlock()

	if sink.client == nil {
		// the client could be nil here, so we reconnect
		if err := sink.connectRiemannClient(); err != nil {
			glog.Warningf("Riemann sink not connected: %v", err)
			return
		}
	}

	var events []riemanngo.Event

	for _, metricSet := range dataBatch.MetricSets {
		host := metricSet.Labels[core.LabelHostname.Key]
		for metricName, metricValue := range metricSet.MetricValues {
			if value := metricValue.GetValue(); value != nil {
				timestamp := dataBatch.Timestamp.Unix()
				// creates an event and add it to dataEvent
				events = appendEvent(events, sink, host, metricName, value, metricSet.Labels, timestamp)
			}
		}
		for _, metric := range metricSet.LabeledMetrics {
			if value := metric.GetValue(); value != nil {
				labels := make(map[string]string)
				for k, v := range metricSet.Labels {
					labels[k] = v
				}
				for k, v := range metric.Labels {
					labels[k] = v
				}
				timestamp := dataBatch.Timestamp.Unix()
				// creates an event and add it to dataEvent
				appendEvent(events, sink, host, metric.Name, value, labels, timestamp)
			}
		}
	}
	// Send events to Riemann if events is not empty
	if len(events) > 0 {
		sink.sendData(events)
	}
}

// Send Events to Riemann using the client from the sink.
func (sink *riemannSink) sendData(events []riemanngo.Event) {
	// do nothing if we are not connected
	if sink.client == nil {
		glog.Warningf("Riemann sink not connected")
		return
	}
	start := time.Now()
	_, err := riemanngo.SendEvents(sink.client, &events)
	end := time.Now()
	if err == nil {
		glog.V(4).Infof("Exported %d events to riemann in %s", len(events), end.Sub(start))
	} else {
		glog.Warningf("There were errors sending events to Riemman, forcing reconnection. Error : %+v", err)
		// client will reconnect later
		sink.client = nil
	}
}
