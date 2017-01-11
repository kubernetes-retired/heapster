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

package kafka

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	event_core "k8s.io/heapster/events/core"
	kube_api "k8s.io/kubernetes/pkg/api"
	kube_api_unversioned "k8s.io/kubernetes/pkg/api/unversioned"
)

type fakeKafkaClient struct {
	points []KafkaSinkPoint
}

type fakeKafkaSink struct {
	event_core.EventSink
	fakeClient *fakeKafkaClient
}

func NewFakeKafkaClient() *fakeKafkaClient {
	return &fakeKafkaClient{[]KafkaSinkPoint{}}
}

func (client *fakeKafkaClient) ProduceKafkaMessage(msgData interface{}) error {
	if point, ok := msgData.(KafkaSinkPoint); ok {
		client.points = append(client.points, point)
	}

	return nil
}

func (client *fakeKafkaClient) Name() string {
	return "Apache Kafka Sink"
}

func (client *fakeKafkaClient) Stop() {
	// nothing needs to be done.
}

// Returns a fake kafka sink.
func NewFakeSink() fakeKafkaSink {
	client := NewFakeKafkaClient()
	return fakeKafkaSink{
		&kafkaSink{
			KafkaClient: client,
		},
		client,
	}
}

func TestStoreDataEmptyInput(t *testing.T) {
	fakeSink := NewFakeSink()
	eventsBatch := event_core.EventBatch{}
	fakeSink.ExportEvents(&eventsBatch)
	assert.Equal(t, 0, len(fakeSink.fakeClient.points))
}

func TestStoreMultipleEventsInput(t *testing.T) {
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
	data := event_core.EventBatch{
		Timestamp: timestamp,
		Events: []*kube_api.Event{
			&event1,
			&event2,
		},
	}
	fakeSink.ExportEvents(&data)
	// expect msg string
	assert.Equal(t, 2, len(fakeSink.fakeClient.points))

}
