// Copyright 2016 Google Inc. All Rights Reserved.
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
	"sync"

	"strings"
	"time"

	riemann_api "github.com/bigdatadev/goryman"
	"github.com/golang/glog"
	riemannCommon "k8s.io/heapster/common/riemann"
	core "k8s.io/heapster/events/core"
	kube_api "k8s.io/kubernetes/pkg/api"
)

// Abstracted for testing: this package works against any client that obeys the
// interface contract exposed by the goryman Riemann client

type RiemannSink struct {
	client riemannCommon.RiemannClient
	config riemannCommon.RiemannConfig
	sync.RWMutex
}

func CreateRiemannSink(uri *url.URL) (core.EventSink, error) {
	var sink, err = riemannCommon.CreateRiemannSink(uri)
	if err != nil {
		return nil, err
	}
	rs := &RiemannSink{
		client: sink.Client,
		config: sink.Config,
	}
	return rs, nil
}

// Return a user-friendly string describing the sink
func (sink *RiemannSink) Name() string {
	return "Riemann Sink"
}

func (sink *RiemannSink) Stop() {
	sink.client.Close()
}

func (sink *RiemannSink) ExportEvents(eventBatch *core.EventBatch) {
	sink.Lock()
	defer sink.Unlock()

	if sink.client == nil {
		var client, err = riemannCommon.SetupRiemannClient(sink.config)
		if err != nil {
			glog.Warningf("Riemann sink not connected: %v", err)
			return
		} else {
			sink.client = client
		}
	}

	var dataEvents []riemann_api.Event

	eventState := func(event *kube_api.Event) string {
		switch event.Type {
		case "Normal":
			return "ok"
		case "Warning":
			return "warning"
		default:
			return "warning"
		}
	}

	appendEvent := func(event *kube_api.Event, timestamp time.Time) {
		riemannEvent := riemann_api.Event{
			Time:        eventBatch.Timestamp.Unix(),
			Service:     strings.Join([]string{event.InvolvedObject.Kind, event.Reason}, "."),
			Host:        event.Source.Host,
			Description: event.Message,
			Attributes: map[string]string{
				"namespace":        event.InvolvedObject.Namespace,
				"uid":              string(event.InvolvedObject.UID),
				"name":             event.InvolvedObject.Name,
				"api-version":      event.InvolvedObject.APIVersion,
				"resource-version": event.InvolvedObject.ResourceVersion,
				"field-path":       event.InvolvedObject.FieldPath,
				"component":        event.Source.Component,
			},
			Metric: riemannCommon.RiemannValue(event.Count),
			Ttl:    sink.config.Ttl,
			State:  eventState(event),
			Tags:   sink.config.Tags,
		}

		dataEvents = append(dataEvents, riemannEvent)
		if len(dataEvents) >= riemannCommon.MaxSendBatchSize {
			sink.sendData(dataEvents)
			dataEvents = nil
		}
	}

	for _, event := range eventBatch.Events {
		appendEvent(event, eventBatch.Timestamp)
	}

	if len(dataEvents) > 0 {
		sink.sendData(dataEvents)
	}
}

func (sink *RiemannSink) sendData(dataEvents []riemann_api.Event) {
	if sink.client == nil {
		return
	}

	start := time.Now()
	errors := 0
	for _, event := range dataEvents {
		glog.V(8).Infof("Sending event to Riemann:  %+v", event)
		var err error
		for try := 0; try < riemannCommon.MaxRetries; try++ {
			err = sink.client.SendEvent(&event)
			if err == nil {
				break
			}
		}
		if err != nil {
			errors++
			glog.V(4).Infof("Failed to send event to Riemann: %+v: %+v", event, err)
		}
	}
	end := time.Now()
	if errors > 0 {
		glog.V(2).Info("There were errors sending events to Riemman, forcing reconnection")
		sink.client.Close()
		sink.client = nil
	}
	glog.V(4).Infof("Exported %d events to riemann in %s", len(dataEvents)-errors, end.Sub(start))
}
