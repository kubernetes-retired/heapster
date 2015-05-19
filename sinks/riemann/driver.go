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
	"fmt"
	riemann_api "github.com/bigdatadev/goryman"
	"github.com/golang/glog"
	"k8s.io/heapster/extpoints"
	sink_api "k8s.io/heapster/sinks/api"
	kube_api "k8s.io/kubernetes/pkg/api"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
)

// The basic internal type for this package; it obeys the sink_api.ExternalSink
// interface

type riemannSink struct {
	client riemannClient
	config riemannConfig
}

type riemannConfig struct {
	host        string
	ttl         float32
	state       string
	tags        []string
	storeEvents bool
}

func init() {
	extpoints.SinkFactories.Register(CreateRiemannSink, "riemann")
}

func CreateRiemannSink(uri *url.URL, _ extpoints.HeapsterConf) ([]sink_api.ExternalSink, error) {
	c := riemannConfig{
		host:        "riemann-heapster:5555",
		ttl:         60.0,
		state:       "",
		tags:        make([]string, 0),
		storeEvents: true,
	}
	if len(uri.Host) > 0 {
		c.host = uri.Host
	}
	options := uri.Query()
	if len(options["ttl"]) > 0 {
		var ttl, err = strconv.ParseFloat(options["ttl"][0], 32)
		if err != nil {
			return nil, err
		}
		c.ttl = float32(ttl)
	}
	if len(options["state"]) > 0 {
		c.state = options["state"][0]
	}
	if len(options["tags"]) > 0 {
		c.tags = options["tags"]
	}
	if len(options["storeEvents"]) > 0 {
		var storeEvents, err = strconv.ParseBool(options["storeEvents"][0])
		if err != nil {
			return nil, err
		}
		c.storeEvents = storeEvents
	}
	glog.V(4).Infof("Riemann sink URI: '%+v', host: '%+v', options: '%+v', ", uri, c.host, options)
	var sink, err = new(func() riemannClient {
		return riemann_api.NewGorymanClient(c.host)
	}, c)
	if err != nil {
		return nil, err
	}
	return []sink_api.ExternalSink{sink}, nil
}

func new(getClient func() riemannClient, conf riemannConfig) (sink_api.ExternalSink, error) {
	var client = getClient()
	var err = client.Connect()
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(client, func(c riemannClient) { c.Close() })
	return &riemannSink{client: client, config: conf}, nil
}

// Abstracted for testing: this package works against any client that obeys the
// interface contract exposed by the goryman Riemann client

type riemannClient interface {
	Connect() error
	Close() error
	SendEvent(*riemann_api.Event) error
}

func (self *riemannSink) kubeEventToRiemannEvent(event kube_api.Event) (*riemann_api.Event, error) {
	glog.V(4).Infof("riemannSink.kubeEventToRiemannEvent(%+v)", event)

	var rv = riemann_api.Event{
		// LastTimestamp: the time at which the most recent occurance of this
		// event was recorded
		Time: event.LastTimestamp.UTC().Unix(),
		// Source: A short, machine-understantable string that gives the
		// component reporting this event
		Host:       event.Source.Host,
		Attributes: event.Labels, //TODO(jfoy) consider event.Annotations as well
		Service:    event.InvolvedObject.Name,
		// Reason: a short, machine-understandable string that gives the reason
		// for this event being generated
		// Message: A human-readable description of the status of this operation.
		Description: event.Reason + ": " + event.Message,
		// Count: the number of times this event has occurred
		Metric: event.Count,
		Ttl:    self.config.ttl,
		State:  self.config.state,
		Tags:   self.config.tags,
	}
	glog.V(4).Infof("Returning Riemann event: %+v", rv)
	return &rv, nil
}

func (self *riemannSink) timeseriesToRiemannEvent(ts sink_api.Timeseries) (*riemann_api.Event, error) {
	glog.V(4).Infof("riemannSink.timeseriesToRiemannEvent(%+v)", ts)

	var service string
	if ts.MetricDescriptor.Units.String() == "" {
		service = ts.Point.Name
	} else {
		service = ts.Point.Name + "_" + ts.MetricDescriptor.Units.String()
	}

	var rv = riemann_api.Event{
		Time:        ts.Point.End.UTC().Unix(),
		Service:     service,
		Host:        ts.Point.Labels[sink_api.LabelHostname.Key],
		Description: ts.MetricDescriptor.Description,
		Attributes:  ts.Point.Labels,
		Metric:      valueToMetric(ts.Point.Value),
		Ttl:         self.config.ttl,
		State:       self.config.state,
		Tags:        self.config.tags,
	}
	glog.V(4).Infof("Returning Riemann event: %+v", rv)
	return &rv, nil
}

func valueToMetric(i interface{}) interface{} {
	value := reflect.ValueOf(i)
	kind := reflect.TypeOf(value.Interface()).Kind()

	switch kind {
	case reflect.Int, reflect.Int64:
		return int(value.Int())
	default:
		return i
	}
}

func (self *riemannSink) sendEvents(events []*riemann_api.Event) error {
	var result error
	for _, event := range events {
		err := self.client.SendEvent(event)
		// FIXME handle multiple errors
		if err != nil {
			glog.Warningf("Failed sending event to Riemann: %+v: %+v", event, err)
			result = err
		}
	}
	return result
}

// Riemann does not pre-register metrics, so Register() is a no-op
func (self *riemannSink) Register(descriptor []sink_api.MetricDescriptor) error { return nil }

// Like Register
func (self *riemannSink) Unregister(metrics []sink_api.MetricDescriptor) error { return nil }

// Send a collection of Timeseries to Riemann
func (self *riemannSink) StoreTimeseries(inputs []sink_api.Timeseries) error {
	// glog.V(3).Infof("riemannSink.StoreTimeseries(%+v)", inputs)

	var events []*riemann_api.Event
	for _, input := range inputs {
		var event, err = self.timeseriesToRiemannEvent(input)
		if err != nil {
			return err
		}
		events = append(events, event)
	}
	return self.sendEvents(events)
}

// Send a collection of Kubernetes Events to Riemann
func (self *riemannSink) StoreEvents(inputs []kube_api.Event) error {
	// glog.V(3).Infof("riemannSink.StoreEvents(%+v)", inputs)
	if !self.config.storeEvents {
		return nil
	}

	var events []*riemann_api.Event
	for _, input := range inputs {
		var event, err = self.kubeEventToRiemannEvent(input)
		if err != nil {
			return err
		}
		events = append(events, event)
	}
	return self.sendEvents(events)
}

// Return debug information specific to Riemann
func (self *riemannSink) DebugInfo() string { return fmt.Sprintf("%s", self.client) }

// Return a user-friendly string describing the sink
func (self *riemannSink) Name() string { return "Riemann" }
