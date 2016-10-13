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

package riemann

import (
	"encoding/json"
	"testing"
	"time"

	riemann_api "github.com/bigdatadev/goryman"
	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/events/core"
	kube_api "k8s.io/kubernetes/pkg/api"
	kube_api_unversioned "k8s.io/kubernetes/pkg/api/unversioned"
)

type eventSendToRiemann struct {
	event string
}

type fakeRiemannClient struct {
	events []eventSendToRiemann
}

type fakeRiemannSink struct {
	core.EventSink
	fakeRiemannClient *fakeRiemannClient
}

func NewFakeRiemannClient() *fakeRiemannClient {
	return &fakeRiemannClient{[]eventSendToRiemann{}}
}

func (client *fakeRiemannClient) Connect() error {
	return nil
}

func (client *fakeRiemannClient) Close() error {
	return nil
}

func (client *fakeRiemannClient) SendEvent(e *riemann_api.Event) error {
	eventsJson, _ := json.Marshal(e)
	client.events = append(client.events, eventSendToRiemann{event: string(eventsJson)})
	return nil
}

// Returns a fake kafka sink.
func NewFakeSink() fakeRiemannSink {
	riemannClient := NewFakeRiemannClient()
	c := riemannConfig{
		host:  "riemann-heapster:5555",
		ttl:   60.0,
		state: "",
		tags:  make([]string, 0),
	}

	return fakeRiemannSink{
		&riemannSink{
			client: riemannClient,
			config: c,
		},
		riemannClient,
	}
}

func TestStoreDataEmptyInput(t *testing.T) {
	fakeSink := NewFakeSink()
	dataBatch := core.EventBatch{}
	fakeSink.ExportEvents(&dataBatch)
	assert.Equal(t, 0, len(fakeSink.fakeRiemannClient.events))
}

func TestStoreMultipleDataInput(t *testing.T) {
	fakeSink := NewFakeSink()
	timestamp := time.Now()
	now := time.Now()

	event1 := kube_api.Event{
		Message:        "event1",
		Count:          100,
		LastTimestamp:  kube_api_unversioned.NewTime(now),
		FirstTimestamp: kube_api_unversioned.NewTime(now),
	}
	event2 := kube_api.Event{
		Message:        "event2",
		Count:          101,
		LastTimestamp:  kube_api_unversioned.NewTime(now),
		FirstTimestamp: kube_api_unversioned.NewTime(now),
	}
	data := core.EventBatch{
		Timestamp: timestamp,
		Events: []*kube_api.Event{
			&event1,
			&event2,
		},
	}
	fakeSink.ExportEvents(&data)
	// expect msg string
	assert.Equal(t, 2, len(fakeSink.fakeRiemannClient.events))
}
