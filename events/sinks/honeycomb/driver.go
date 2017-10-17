// Copyright 2017 Google Inc. All Rights Reserved.
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

package honeycomb

import (
	"net/url"
	"sync"

	"github.com/golang/glog"
	kube_api "k8s.io/api/core/v1"
	honeycomb_common "k8s.io/heapster/common/honeycomb"
	event_core "k8s.io/heapster/events/core"
)

type honeycombSink struct {
	client honeycomb_common.Client
	sync.Mutex
}

type exportedData struct {
	Namespace       string `json:"namespace"`
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	SubObject       string `json:"subobject"`
	SourceComponent string `json:"source.component"`
	SourceHost      string `json:"source.host"`
	Count           int32  `json:"count"`
	Type            string `json:"type"`
	Reason          string `json:"reason"`
	Message         string `json:"message"`
}

func getExportedData(e *kube_api.Event) *exportedData {
	return &exportedData{
		Namespace:       e.InvolvedObject.Namespace,
		Kind:            e.InvolvedObject.Kind,
		Name:            e.InvolvedObject.Name,
		SubObject:       e.InvolvedObject.FieldPath,
		SourceComponent: e.Source.Component,
		SourceHost:      e.Source.Host,
		Count:           e.Count,
		Reason:          e.Reason,
		Type:            e.Type,
		Message:         e.Message,
	}
}

func (sink *honeycombSink) ExportEvents(eventBatch *event_core.EventBatch) {
	sink.Lock()
	defer sink.Unlock()
	exportedBatch := make(honeycomb_common.Batch, len(eventBatch.Events))
	for i, event := range eventBatch.Events {
		data := getExportedData(event)
		exportedBatch[i] = &honeycomb_common.BatchPoint{
			Data:      data,
			Timestamp: event.LastTimestamp.UTC(),
		}
	}
	err := sink.client.SendBatch(exportedBatch)
	if err != nil {
		glog.Warningf("Failed to send event: %v", err)
		return
	}
}

func (sink *honeycombSink) Stop() {}

func (sink *honeycombSink) Name() string {
	return "Honeycomb Sink"
}

func NewHoneycombSink(uri *url.URL) (event_core.EventSink, error) {
	client, err := honeycomb_common.NewClient(uri)
	if err != nil {
		return nil, err
	}
	sink := &honeycombSink{
		client: client,
	}

	return sink, nil

}
