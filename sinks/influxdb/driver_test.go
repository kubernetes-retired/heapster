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
	"testing"

	influxdb "github.com/influxdb/influxdb/client"
	"github.com/stretchr/testify/assert"
	sink_api "k8s.io/heapster/sinks/api"
	kube_api "k8s.io/kubernetes/pkg/api"
	kube_api_unv "k8s.io/kubernetes/pkg/api/unversioned"
)

type capturedWriteCall struct {
	series        []*influxdb.Series
	timePrecision influxdb.TimePrecision
}

type fakeInfluxDBClient struct {
	capturedWriteCalls []capturedWriteCall
}

func NewFakeInfluxDBClient() *fakeInfluxDBClient {
	return &fakeInfluxDBClient{[]capturedWriteCall{}}
}

func (sink *fakeInfluxDBClient) WriteSeriesWithTimePrecision(series []*influxdb.Series, timePrecision influxdb.TimePrecision) error {
	sink.capturedWriteCalls = append(sink.capturedWriteCalls, capturedWriteCall{series, timePrecision})
	return nil
}

func (sink *fakeInfluxDBClient) GetDatabaseList() ([]map[string]interface{}, error) {
	// No-op
	return nil, nil
}

func (sink *fakeInfluxDBClient) CreateDatabase(name string) error {
	// No-op
	return nil
}

func (sink *fakeInfluxDBClient) DisableCompression() {
	// No-op
}

type fakeInfluxDBSink struct {
	sink_api.ExternalSink
	fakeClient *fakeInfluxDBClient
}

// Returns a fake influxdb sink.
func NewFakeSink(avoidColumns bool) fakeInfluxDBSink {
	client := NewFakeInfluxDBClient()
	return fakeInfluxDBSink{
		&influxdbSink{
			client: client,
			seqNum: newMetricSequenceNum(),
			c: config{
				host:         "hostname",
				dbName:       "databaseName",
				avoidColumns: avoidColumns,
			},
		},
		client,
	}
}

func TestStoreEventsNilInput(t *testing.T) {
	// Arrange
	fakeSink := NewFakeSink(false /* avoidColumns */)

	// Act
	err := fakeSink.StoreEvents(nil /*events*/)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls) /* actual */)
}

func TestStoreEventsEmptyInput(t *testing.T) {
	// Arrange
	fakeSink := NewFakeSink(false /* avoidColumns */)

	// Act
	err := fakeSink.StoreEvents([]kube_api.Event{})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls) /* actual */)
}

func TestStoreEventsSingleEventInput(t *testing.T) {
	// Arrange
	fakeSink := NewFakeSink(false /* avoidColumns */)
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

	// Act
	err := fakeSink.StoreEvents(events)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls) /* actual */)
	assert.Equal(t, influxdb.Millisecond /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].timePrecision /* actual */)
	assert.Equal(t, 1 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls[0].series) /* actual */)
	assert.Equal(t, eventsSeriesName /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Name /* actual */)
	assert.Equal(t, 6 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls[0].series[0].Columns) /* actual */)
	assert.Equal(t, 1 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points) /* actual */)
	assert.Equal(t, eventTime.Unix() /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][0] /* actual */)           // Column 0 - time
	assert.Equal(t, uint64(0xcbf29ce484222325) /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][1] /* actual */) // Column 1 - sequence_number
	assert.Equal(t, "" /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][2] /* actual */)                         // Column 2 - pod_id
	assert.Equal(t, "" /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][3] /* actual */)                         // Column 3 - pod_id
	assert.Equal(t, eventSourceHostname /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][4] /* actual */)        // Column 4 - pod_id
	assert.Contains(t, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][5], eventReason)                                         // Column 5 - value
}

func TestStoreEventsMultipleEventsInput(t *testing.T) {
	// Arrange
	fakeSink := NewFakeSink(false /* avoidColumns */)
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

	// Act
	err := fakeSink.StoreEvents(events)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls) /* actual */)
	assert.Equal(t, influxdb.Millisecond /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].timePrecision /* actual */)
	assert.Equal(t, 1 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls[0].series) /* actual */)
	assert.Equal(t, eventsSeriesName /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Name /* actual */)
	assert.Equal(t, 6 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls[0].series[0].Columns) /* actual */)
	assert.Equal(t, 2 /* expected */, len(fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points) /* actual */)
	assert.Equal(t, event1Time.Unix() /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][0] /* actual */)          // Column 0 - time
	assert.Equal(t, uint64(0xcbf29ce484222325) /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][1] /* actual */) // Column 1 - sequence_number
	assert.Equal(t, "" /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][2] /* actual */)                         // Column 2 - pod_id
	assert.Equal(t, "" /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][3] /* actual */)                         // Column 3 - pod_id
	assert.Equal(t, event1SourceHostname /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][4] /* actual */)       // Column 4 - pod_id
	assert.Contains(t, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[0][5], event1Reason)                                        // Column 5 - value
	assert.Equal(t, event2Time.Unix() /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[1][0] /* actual */)          // Column 0 - time
	assert.Equal(t, uint64(0xcbf29ce484222325) /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[1][1] /* actual */) // Column 1 - sequence_number
	assert.Equal(t, "" /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[1][2] /* actual */)                         // Column 2 - pod_id
	assert.Equal(t, "" /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[1][3] /* actual */)                         // Column 3 - pod_id
	assert.Equal(t, event2SourceHostname /* expected */, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[1][4] /* actual */)       // Column 4 - pod_id
	assert.Contains(t, fakeSink.fakeClient.capturedWriteCalls[0].series[0].Points[1][5], event2Reason)                                        // Column 5 - value

}
