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

package plugin

import (
	"encoding/json"
	"net/url"
	"sync"
	"time"

	"context"

	"github.com/golang/glog"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	kube_api "k8s.io/api/core/v1"
	plugin_common "k8s.io/heapster/common/plugin"
	event_core "k8s.io/heapster/events/core"
	proto "k8s.io/heapster/events/sinks/plugin/proto"
	"k8s.io/heapster/metrics/core"
)

func init() {
	plugin_common.PluginMap["events"] = &EventSinkPlugin{}
}

type PluginSinkPoint struct {
	EventValue     string
	EventTimestamp time.Time
	EventTags      map[string]string
}

type EventSink interface {
	WriteEventsPoint(point *PluginSinkPoint) error
}

type EventSinkPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl EventSink
}

func (p *EventSinkPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterEventPluginServer(s, &GRPCServer{
		Impl: p.Impl,
	})
	return nil
}

func (p *EventSinkPlugin) GRPCClient(context context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{
		client: proto.NewEventPluginClient(c),
	}, nil
}

type GoPluginSink struct {
	plugin_common.GoPluginSink
	EventSink
	sync.RWMutex
}

func getEventValue(event *kube_api.Event) (string, error) {
	// TODO: check whether indenting is required.
	bytes, err := json.MarshalIndent(event, "", " ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func eventToPoint(event *kube_api.Event) (*PluginSinkPoint, error) {
	value, err := getEventValue(event)
	if err != nil {
		return nil, err
	}
	point := PluginSinkPoint{
		EventTimestamp: event.LastTimestamp.Time.UTC(),
		EventValue:     value,
		EventTags: map[string]string{
			"eventID": string(event.UID),
		},
	}
	if event.InvolvedObject.Kind == "Pod" {
		point.EventTags[core.LabelPodId.Key] = string(event.InvolvedObject.UID)
		point.EventTags[core.LabelPodName.Key] = event.InvolvedObject.Name
	}
	point.EventTags[core.LabelHostname.Key] = event.Source.Host
	return &point, nil
}

func (sink *GoPluginSink) ExportEvents(eventBatch *event_core.EventBatch) {
	sink.Lock()
	defer sink.Unlock()

	for _, event := range eventBatch.Events {
		point, err := eventToPoint(event)
		if err != nil {
			glog.Warningf("Failed to convert event to point: %v", err)
			continue
		}

		err = sink.WriteEventsPoint(point)
		if err != nil {
			glog.Errorf("Failed to produce event message: %s", err)
		}
	}
}

func NewGoPluginSink(uri *url.URL) (event_core.EventSink, error) {
	client, err := plugin_common.NewGoPluginClient(uri)
	if err != nil {
		return nil, err
	}
	sink, err := client.Plugin("events")
	if err != nil {
		return nil, err
	}
	return &GoPluginSink{
		GoPluginSink: client,
		EventSink:    sink.(EventSink),
	}, nil
}

var _ plugin.GRPCPlugin = &EventSinkPlugin{}
