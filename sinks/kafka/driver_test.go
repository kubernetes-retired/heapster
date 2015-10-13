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
	"encoding/json"
	"testing"
	"time"

	"github.com/optiopay/kafka/proto"
	"github.com/stretchr/testify/assert"
	sink_api "k8s.io/heapster/sinks/api"
	kube_api "k8s.io/kubernetes/pkg/api"
	kube_api_unv "k8s.io/kubernetes/pkg/api/unversioned"
)

type msgProducedToKafka struct {
	message string
}

type fakeKafkaProducer struct {
	msgs []msgProducedToKafka
}

type fakeKafkaSink struct {
	sink_api.ExternalSink
	fakeProducer *fakeKafkaProducer
}

func NewFakeKafkaProducer() *fakeKafkaProducer {
	return &fakeKafkaProducer{[]msgProducedToKafka{}}
}

func (producer *fakeKafkaProducer) Produce(topic string, partition int32, messages ...*proto.Message) (offset int64, err error) {
	for index := range messages {
		producer.msgs = append(producer.msgs, msgProducedToKafka{string(messages[index].Value)})
	}
	return 0, nil
}

// Returns a fake kafka sink.
func NewFakeSink() fakeKafkaSink {
	producer := NewFakeKafkaProducer()
	fakeTimeSeriesTopic := "kafkaTime-test-topic"
	fakeEventsTopic := "kafkaEvent-test-topic"
	fakesinkBrokerHosts := make([]string, 2)
	return fakeKafkaSink{
		&kafkaSink{
			producer,
			fakeTimeSeriesTopic,
			fakeEventsTopic,
			fakesinkBrokerHosts,
		},
		producer,
	}
}

func TestStoreEventsNilInput(t *testing.T) {
	fakeSink := NewFakeSink()
	err := fakeSink.StoreEvents(nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(fakeSink.fakeProducer.msgs))
}

func TestStoreEventsEmptyInput(t *testing.T) {
	fakeSink := NewFakeSink()
	err := fakeSink.StoreEvents([]kube_api.Event{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(fakeSink.fakeProducer.msgs))
}

func TestStoreEventsSingleEventInput(t *testing.T) {
	fakeSink := NewFakeSink()
	eventTime := kube_api_unv.Unix(12345, 0)
	eventSourceHostname := "event1HostName"
	eventReason := "event1"
	events := []kube_api.Event{
		{
			Reason:        eventReason,
			LastTimestamp: eventTime,
			Source: kube_api.EventSource{
				Host: eventSourceHostname,
			},
		},
	}
	//expect msg string
	eventsJson, _ := json.Marshal(events[0])
	err := fakeSink.StoreEvents(events)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fakeSink.fakeProducer.msgs))
	assert.Equal(t, eventsJson, fakeSink.fakeProducer.msgs[0].message)
}

func TestStoreEventsMultipleEventsInput(t *testing.T) {
	fakeSink := NewFakeSink()
	event1Time := kube_api_unv.Unix(12345, 0)
	event2Time := kube_api_unv.Unix(12366, 0)
	event1SourceHostname := "event1HostName"
	event2SourceHostname := "event2HostName"
	event1Reason := "event1"
	event2Reason := "event2"
	events := []kube_api.Event{
		{
			Reason:        event1Reason,
			LastTimestamp: event1Time,
			Source: kube_api.EventSource{
				Host: event1SourceHostname,
			},
		},
		{
			Reason:        event2Reason,
			LastTimestamp: event2Time,
			Source: kube_api.EventSource{
				Host: event2SourceHostname,
			},
		},
	}
	err := fakeSink.StoreEvents(events)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fakeSink.fakeProducer.msgs))
	events0Json, _ := json.Marshal(events[0])
	events1Json, _ := json.Marshal(events[1])
	assert.Equal(t, string(events0Json), fakeSink.fakeProducer.msgs[0].message)
	assert.Equal(t, string(events1Json), fakeSink.fakeProducer.msgs[1].message)
}

func TestStoreTimeseriesNilInput(t *testing.T) {
	fakeSink := NewFakeSink()
	err := fakeSink.StoreTimeseries(nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(fakeSink.fakeProducer.msgs))
}

func TestStoreTimeseriesEmptyInput(t *testing.T) {
	fakeSink := NewFakeSink()
	err := fakeSink.StoreTimeseries([]sink_api.Timeseries{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(fakeSink.fakeProducer.msgs))
}

func TestStoreTimeseriesSingleTimeserieInput(t *testing.T) {
	fakeSink := NewFakeSink()

	smd := sink_api.MetricDescriptor{
		ValueType: sink_api.ValueInt64,
		Type:      sink_api.MetricCumulative,
	}

	l := make(map[string]string)
	l["test"] = "notvisible"
	l[sink_api.LabelHostname.Key] = "localhost"
	l[sink_api.LabelContainerName.Key] = "docker"
	l[sink_api.LabelPodId.Key] = "aaaa-bbbb-cccc-dddd"

	p := sink_api.Point{
		Name:   "test/metric/1",
		Labels: l,
		Start:  time.Now(),
		End:    time.Now(),
		Value:  int64(123456),
	}

	timeseries := []sink_api.Timeseries{
		{
			MetricDescriptor: &smd,
			Point:            &p,
		},
	}

	err := fakeSink.StoreTimeseries(timeseries)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fakeSink.fakeProducer.msgs))
	timeseries1Json, _ := json.Marshal(timeseries[0])
	assert.Equal(t, string(timeseries1Json), fakeSink.fakeProducer.msgs[0].message)
}

func TestStoreTimeseriesMultipleTimeseriesInput(t *testing.T) {
	fakeSink := NewFakeSink()

	smd := sink_api.MetricDescriptor{
		ValueType: sink_api.ValueInt64,
		Type:      sink_api.MetricCumulative,
	}

	l := make(map[string]string)
	l["test"] = "notvisible"
	l[sink_api.LabelHostname.Key] = "localhost"
	l[sink_api.LabelContainerName.Key] = "docker"
	l[sink_api.LabelPodId.Key] = "aaaa-bbbb-cccc-dddd"

	p1 := sink_api.Point{
		Name:   "test/metric/1",
		Labels: l,
		Start:  time.Now(),
		End:    time.Now(),
		Value:  int64(123456),
	}

	p2 := sink_api.Point{
		Name:   "test/metric/1",
		Labels: l,
		Start:  time.Now(),
		End:    time.Now(),
		Value:  int64(123456),
	}

	timeseries := []sink_api.Timeseries{
		{
			MetricDescriptor: &smd,
			Point:            &p1,
		},
		{
			MetricDescriptor: &smd,
			Point:            &p2,
		},
	}

	err := fakeSink.StoreTimeseries(timeseries)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fakeSink.fakeProducer.msgs))
	timeseries0Json, _ := json.Marshal(timeseries[0])
	timeseries1Json, _ := json.Marshal(timeseries[1])
	assert.Equal(t, string(timeseries0Json), fakeSink.fakeProducer.msgs[0].message)
	assert.Equal(t, string(timeseries1Json), fakeSink.fakeProducer.msgs[1].message)
}
