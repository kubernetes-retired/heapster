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

package riemann

import (
	"strconv"
	"testing"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/riemann/riemann-go-client/proto"
	"github.com/stretchr/testify/assert"
	kube_api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	riemannCommon "k8s.io/heapster/common/riemann"
	"k8s.io/heapster/events/core"
)

type fakeRiemannClient struct {
	events []proto.Event
}

type fakeRiemannSink struct {
	core.EventSink
	fakeRiemannClient *fakeRiemannClient
}

func NewFakeRiemannClient() *fakeRiemannClient {
	return &fakeRiemannClient{[]proto.Event{}}
}

func (client *fakeRiemannClient) Connect(timeout int32) error {
	return nil
}

func (client *fakeRiemannClient) Close() error {
	return nil
}

func (client *fakeRiemannClient) Send(e *proto.Msg) (*proto.Msg, error) {
	msg := &proto.Msg{Ok: pb.Bool(true)}
	for _, event := range e.Events {
		client.events = append(client.events, *event)
	}
	// always returns a Ok msg
	return msg, nil
}

// Returns a fake Riemann sink.
func NewFakeSink() fakeRiemannSink {
	riemannClient := NewFakeRiemannClient()
	c := riemannCommon.RiemannConfig{
		Host:      "riemann-heapster:5555",
		Ttl:       60.0,
		State:     "",
		Tags:      []string{"heapster"},
		BatchSize: 1000,
	}

	return fakeRiemannSink{
		&RiemannSink{
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

	data := core.EventBatch{
		Timestamp: timestamp,
		Events: []*kube_api.Event{
			{
				Message: "event1",
				Type:    "Normal",
				Count:   100,
				Source: kube_api.EventSource{
					Host:      "riemann",
					Component: "component",
				},
				LastTimestamp:  metav1.NewTime(timestamp),
				FirstTimestamp: metav1.NewTime(timestamp),
			},
			{
				Message: "event2",
				Type:    "Warning",
				Count:   101,
				Source: kube_api.EventSource{
					Host:      "riemann",
					Component: "component",
				},
				LastTimestamp:  metav1.NewTime(timestamp),
				FirstTimestamp: metav1.NewTime(timestamp),
			},
			{
				Message: "event3",
				Count:   102,
				Source: kube_api.EventSource{
					Host:      "riemann1",
					Component: "component1",
				},
				Reason: "because",
				InvolvedObject: kube_api.ObjectReference{
					Kind:            "kind",
					Namespace:       "kube-system",
					Name:            "objectname",
					UID:             "uuid",
					APIVersion:      "v1",
					ResourceVersion: "v2",
					FieldPath:       "/foo",
				},
			},
		},
	}
	fakeSink.ExportEvents(&data)
	// expect msg string
	assert.Equal(t, 3, len(fakeSink.fakeRiemannClient.events))
	timeValue := timestamp.Unix()
	var expectedEvents = []*proto.Event{
		{
			Host:         pb.String("riemann"),
			Time:         pb.Int64(timeValue),
			Ttl:          pb.Float32(60),
			MetricSint64: pb.Int64(100),
			Service:      pb.String("."),
			Description:  pb.String("event1"),
			State:        pb.String("ok"),
			Tags:         []string{"heapster"},
			Attributes: []*proto.Attribute{
				{
					Key:   pb.String("api-version"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("component"),
					Value: pb.String("component"),
				},
				{
					Key:   pb.String("field-path"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("first-timestamp"),
					Value: pb.String(strconv.FormatInt(timeValue, 10)),
				},
				{
					Key:   pb.String("last-timestamp"),
					Value: pb.String(strconv.FormatInt(timeValue, 10)),
				},
				{
					Key:   pb.String("name"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("namespace"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("resource-version"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("uid"),
					Value: pb.String(""),
				},
			},
		},
		{
			Host:         pb.String("riemann"),
			Time:         pb.Int64(timeValue),
			Ttl:          pb.Float32(60),
			MetricSint64: pb.Int64(101),
			Service:      pb.String("."),
			Description:  pb.String("event2"),
			State:        pb.String("warning"),
			Tags:         []string{"heapster"},
			Attributes: []*proto.Attribute{
				{
					Key:   pb.String("api-version"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("component"),
					Value: pb.String("component"),
				},
				{
					Key:   pb.String("field-path"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("first-timestamp"),
					Value: pb.String(strconv.FormatInt(timeValue, 10)),
				},
				{
					Key:   pb.String("last-timestamp"),
					Value: pb.String(strconv.FormatInt(timeValue, 10)),
				},
				{
					Key:   pb.String("name"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("namespace"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("resource-version"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("uid"),
					Value: pb.String(""),
				},
			},
		},
		{
			Host:         pb.String("riemann1"),
			Time:         pb.Int64(timeValue),
			Ttl:          pb.Float32(60),
			MetricSint64: pb.Int64(102),
			Service:      pb.String("kind.because"),
			Description:  pb.String("event3"),
			State:        pb.String("warning"),
			Tags:         []string{"heapster"},
			Attributes: []*proto.Attribute{
				{
					Key:   pb.String("api-version"),
					Value: pb.String("v1"),
				},
				{
					Key:   pb.String("component"),
					Value: pb.String("component1"),
				},
				{
					Key:   pb.String("field-path"),
					Value: pb.String("/foo"),
				},
				{
					Key:   pb.String("first-timestamp"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("last-timestamp"),
					Value: pb.String(""),
				},
				{
					Key:   pb.String("name"),
					Value: pb.String("objectname"),
				},
				{
					Key:   pb.String("namespace"),
					Value: pb.String("kube-system"),
				},
				{
					Key:   pb.String("resource-version"),
					Value: pb.String("v2"),
				},
				{
					Key:   pb.String("uid"),
					Value: pb.String("uuid"),
				},
			},
		},
	}
	for _, expectedEvent := range expectedEvents {
		found := false
		for _, sinkEvent := range fakeSink.fakeRiemannClient.events {
			if pb.Equal(expectedEvent, &sinkEvent) {
				found = true
			}
		}
		if !found {
			t.Error("Error, event not found in sink")
		}
	}
}
