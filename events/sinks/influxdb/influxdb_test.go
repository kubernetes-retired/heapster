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
	"time"

	"net/http/httptest"
	"net/url"

	influxdb "github.com/influxdb/influxdb/client"
	"github.com/stretchr/testify/assert"
	influxdb_common "k8s.io/heapster/common/influxdb"
	"k8s.io/heapster/events/core"
	kube_api "k8s.io/kubernetes/pkg/api"
	kube_api_unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/util"
)

type pointSavedToInfluxdb struct {
	ponit influxdb.Point
}

type fakeInfluxDBClient struct {
	pnts []pointSavedToInfluxdb
}

type fakeInfluxDBSink struct {
	core.EventSink
	fakeInfluxDbClient *fakeInfluxDBClient
}

func NewFakeInfluxDBClient() *fakeInfluxDBClient {
	return &fakeInfluxDBClient{[]pointSavedToInfluxdb{}}
}

func (client *fakeInfluxDBClient) Write(bps influxdb.BatchPoints) (*influxdb.Response, error) {
	for _, pnt := range bps.Points {
		client.pnts = append(client.pnts, pointSavedToInfluxdb{pnt})
	}
	return nil, nil
}

func (client *fakeInfluxDBClient) Query(influxdb.Query) (*influxdb.Response, error) {
	return nil, nil
}

func (client *fakeInfluxDBClient) Ping() (time.Duration, string, error) {
	return 0, "", nil
}

// Returns a fake influxdb sink.
func NewFakeSink() fakeInfluxDBSink {
	client := NewFakeInfluxDBClient()
	config := influxdb_common.InfluxdbConfig{
		User:     "root",
		Password: "root",
		Host:     "localhost:8086",
		DbName:   "k8s",
		Secure:   false,
	}
	return fakeInfluxDBSink{
		&influxdbSink{
			client: client,
			c:      config,
		},
		client,
	}
}

func TestStoreDataEmptyInput(t *testing.T) {
	fakeSink := NewFakeSink()
	eventBatch := core.EventBatch{}
	fakeSink.ExportEvents(&eventBatch)
	assert.Equal(t, 0, len(fakeSink.fakeInfluxDbClient.pnts))
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
	assert.Equal(t, 2, len(fakeSink.fakeInfluxDbClient.pnts))
}

func TestCreateInfluxdbSink(t *testing.T) {
	handler := util.FakeHandler{
		StatusCode:   200,
		RequestBody:  "",
		ResponseBody: "",
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()

	stubInfluxDBUrl, err := url.Parse(server.URL)
	assert.NoError(t, err)

	//create influxdb sink
	sink, err := CreateInfluxdbSink(stubInfluxDBUrl)
	assert.NoError(t, err)

	//check sink name
	assert.Equal(t, sink.Name(), "InfluxDB Sink")
}
