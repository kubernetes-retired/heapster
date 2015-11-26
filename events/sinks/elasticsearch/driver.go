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
	"fmt"
	"net/url"
	"sync"
	"time"

	"encoding/json"
	"github.com/golang/glog"
	"github.com/olivere/elastic"
	"github.com/pborman/uuid"
	"k8s.io/heapster/metrics/core"
	event_core "k8s.io/heapster/events/core"
	kube_api "k8s.io/kubernetes/pkg/api"
)

const (
	eventSeriesIndex = "heapster-metrics"
	typeName        = "k8s-heapster"
)

// LimitFunc is a pluggable function to enforce limits on the object
type SaveDataFunc func(esClient *elastic.Client, indexName string, typeName string, sinkData interface{}) error

type elasticSearchSink struct {
	esClient         *elastic.Client
	saveDataFunc     SaveDataFunc
	eventSeriesIndex string
	needAuthen       bool
	esUserName       string
	esUserSecret     string
	esNodes          []string
	sync.RWMutex
}

type EsSinkPoint struct {
	EventValue     interface{}
	EventTimestamp time.Time
	EventTags      map[string]string
}

// Generate point value for event
func getEventValue(event *kube_api.Event) (string, error) {
	// TODO: check whether indenting is required.
	bytes, err := json.MarshalIndent(event, "", " ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func eventToPoint(event *kube_api.Event) (*EsSinkPoint, error) {
	value, err := getEventValue(event)
	if err != nil {
		return nil, err
	}

	point := EsSinkPoint{
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

func (sink *elasticSearchSink) ExportEvents(eventBatch *event_core.EventBatch) {
	sink.Lock()
	defer sink.Unlock()

	for _, event := range eventBatch.Events {
		point, err := eventToPoint(event)
		if err != nil {
			glog.Warningf("Failed to convert event to point: %v", err)
		}
		sink.saveDataFunc(sink.esClient, sink.eventSeriesIndex, typeName, point)
	}
}

// SaveDataIntoES save metrics and events to ES by using ES client
func SaveDataIntoES(esClient *elastic.Client, indexName string, typeName string, sinkData interface{}) error {
	if indexName == "" || typeName == "" || sinkData == nil {
		return nil
	}
	// Use the IndexExists service to check if a specified index exists.
	exists, err := esClient.IndexExists(indexName).Do()
	if err != nil {
		return err
	}
	if !exists {
		// Create a new index.
		createIndex, err := esClient.CreateIndex(indexName).Do()
		if err != nil {
			return err
		}
		if !createIndex.Acknowledged {
			return fmt.Errorf("failed to create Index in ES cluster: %s", err)
		}
	}
	indexID := uuid.NewUUID()
	_, err = esClient.Index().
		Index(indexName).
		Type(typeName).
		Id(string(indexID)).
		BodyJson(sinkData).
		Do()
	if err != nil {
		return err
	}
	return nil
}

func (sink *elasticSearchSink) Name() string {
	return "ElasticSearch Sink"
}

func (sink *elasticSearchSink) Stop() {
	// nothing needs to be done.
}

func NewElasticSearchSink(uri *url.URL) (event_core.EventSink, error) {

	var esSink elasticSearchSink
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parser url's query string: %s", err)
	}

	//set the index for eventSeries,the default value is "eventIndex"
	esSink.eventSeriesIndex = eventSeriesIndex
	if len(opts["eventSeriesIndex"]) > 0 {
		esSink.eventSeriesIndex = opts["eventSeriesIndex"][0]
	}

	//If the ES cluster needs authentication, the username and secret
	//should be set in sink config.Else, set the Authenticate flag to false
	esSink.needAuthen = false
	if len(opts["esUserName"]) > 0 && len(opts["esUserSecret"]) > 0 {
		esSink.eventSeriesIndex = opts["esUserName"][0]
		esSink.needAuthen = true
	}

	//set the URL endpoints of the ES's nodes. Notice that
	// when sniffing is enabled, these URLs are used to initially sniff the
	// cluster on startup.
	if len(opts["nodes"]) < 1 {
		return nil, fmt.Errorf("There is no node assigned for connecting ES cluster")
	}
	esSink.esNodes = append(esSink.esNodes, opts["nodes"]...)
	glog.V(2).Infof("initializing elasticsearch sink with ES's nodes - %v", esSink.esNodes)

	var client *(elastic.Client)
	if esSink.needAuthen == false {
		client, err = elastic.NewClient(elastic.SetURL(esSink.esNodes...))
	} else {
		client, err = elastic.NewClient(elastic.SetBasicAuth(esSink.esUserName, esSink.esUserSecret), elastic.SetURL(esSink.esNodes...))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create ElasticSearch client: %v", err)
	}
	esSink.esClient = client
	esSink.saveDataFunc = SaveDataIntoES

	glog.V(2).Infof("elasticsearch sink setup successfully")
	return &esSink, nil
}
