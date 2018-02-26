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

package honeycomb

import (
	"testing"
	"time"

	"net/url"

	"github.com/stretchr/testify/assert"
	kube_api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	honeycomb_common "k8s.io/heapster/common/honeycomb"
	"k8s.io/heapster/events/core"
)

type fakeHoneycombEventSink struct {
	core.EventSink
	fakeDbClient *honeycomb_common.FakeHoneycombClient
}

func NewFakeSink() fakeHoneycombEventSink {
	fakeClient := honeycomb_common.NewFakeHoneycombClient()
	return fakeHoneycombEventSink{
		&honeycombSink{
			client: fakeClient,
		},
		fakeClient,
	}
}

func TestStoreDataEmptyInput(t *testing.T) {
	fakeSink := NewFakeSink()
	eventBatch := core.EventBatch{}
	fakeSink.ExportEvents(&eventBatch)
	assert.Equal(t, 0, len(fakeSink.fakeDbClient.BatchPoints))
}

func TestStoreMultipleDataInput(t *testing.T) {
	fakeSink := NewFakeSink()
	timestamp := time.Now()

	now := time.Now()
	event1 := kube_api.Event{
		Message:        "event1",
		Count:          100,
		LastTimestamp:  metav1.NewTime(now),
		FirstTimestamp: metav1.NewTime(now),
	}

	event2 := kube_api.Event{
		Message:        "event2",
		Count:          101,
		LastTimestamp:  metav1.NewTime(now),
		FirstTimestamp: metav1.NewTime(now),
	}

	data := core.EventBatch{
		Timestamp: timestamp,
		Events: []*kube_api.Event{
			&event1,
			&event2,
		},
	}

	fakeSink.ExportEvents(&data)
	assert.Equal(t, 2, len(fakeSink.fakeDbClient.BatchPoints))
}

func TestCreateHoneycombSink(t *testing.T) {
	stubHoneycombURL, err := url.Parse("?dataset=testdataset&writekey=testwritekey")
	assert.NoError(t, err)

	//create honeycomb sink
	sink, err := NewHoneycombSink(stubHoneycombURL)
	assert.NoError(t, err)

	//check sink name
	assert.Equal(t, sink.Name(), "Honeycomb Sink")
}
