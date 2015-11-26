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

package elasticsearch

import (
	"encoding/json"
	"testing"
	"fmt"
	"time"
	"github.com/olivere/elastic"
	"k8s.io/heapster/events/core"
	"github.com/stretchr/testify/assert"
	kube_api "k8s.io/kubernetes/pkg/api"
	kube_api_unversioned "k8s.io/kubernetes/pkg/api/unversioned"
)

type dataSavedToES struct {
	data string
}

type fakeESSink struct {
	core.EventSink
	savedData []dataSavedToES
}

var FakeESSink fakeESSink

func SaveDataIntoES_Stub(esClient *elastic.Client, indexName string, typeName string, sinkData interface{}) error {
	jsonItems, err := json.Marshal(sinkData)
	if err != nil {
		return fmt.Errorf("failed to transform the items to json : %s", err)
	}
	FakeESSink.savedData = append(FakeESSink.savedData, dataSavedToES{string(jsonItems)})
	return nil
}

// Returns a fake ES sink.
func NewFakeSink() fakeESSink{
	var ESClient elastic.Client
	fakeSinkESNodes := make([]string, 2)
	savedData := make([]dataSavedToES, 0)
	return fakeESSink{
		&elasticSearchSink{
			esClient:        &ESClient,
			saveDataFunc:    SaveDataIntoES_Stub,
			eventSeriesIndex: "heapster-metric-index",
			needAuthen:      false,
			esUserName:      "admin",
			esUserSecret:    "admin",
			esNodes:         fakeSinkESNodes,
		},
		savedData,
	}
}

func TestStoreDataEmptyInput(t *testing.T) {
	fakeSink := NewFakeSink()
	dataBatch := core.EventBatch{}
	fakeSink.ExportEvents(&dataBatch)
	assert.Equal(t, 0, len(fakeSink.savedData))
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

	//expect msg string
	assert.Equal(t, 2, len(FakeESSink.savedData))
	//msgsString := fmt.Sprintf("%s", FakeESSink.savedData)
	//assert.Contains(t, msgsString, "")
	fmt.Println(FakeESSink.savedData)
	FakeESSink = fakeESSink{}
}
