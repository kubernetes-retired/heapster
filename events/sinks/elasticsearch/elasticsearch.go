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
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	es_common "k8s.io/heapster/common/elasticsearch"
	"k8s.io/heapster/events/core"
)

type sendDataFunc func(esClient es_common.ElasticSearchClient, indexName string, typeName string, sinkData interface{}) error

type elasticSearchSink struct {
	client es_common.ElasticSearchClient
	sync.RWMutex
	c            es_common.ElasticSearchConfig
	indexExists  bool
	sendDataFunc sendDataFunc
}

const (
	EventsIndex = "heapster-events"
	typeName    = "k8s-heapster"
)

func (sink *elasticSearchSink) resetConnection() {
	glog.Infof("ElasticSearch connection reset")
	sink.indexExists = false
	sink.client = nil
}

type EsSinkEvent struct {
	EventMessage        string
	EventReason         string
	EventTimestamp      time.Time
	EventCount          int
	EventInvolvedObject interface{}
	EventSource         interface{}
}

func (sink *elasticSearchSink) ExportEvents(eventBatch *core.EventBatch) {
	sink.Lock()
	defer sink.Unlock()

	// Create database if needed
	if err := sink.createIndex(); err != nil {
		glog.Errorf("Failed to create index: %v", err)
		return
	}

	sentEventsErrors := 0
	sentEvents := 0
	start := time.Now()
	for _, event := range eventBatch.Events {
		sinkEvent := EsSinkEvent{
			EventMessage:        event.Message,
			EventReason:         event.Reason,
			EventTimestamp:      event.LastTimestamp.UTC(),
			EventCount:          event.Count,
			EventInvolvedObject: event.InvolvedObject,
			EventSource:         event.Source,
		}

		// TODO: send batch events
		err := sink.sendDataFunc(sink.client, sink.c.EventsIndex, typeName, sinkEvent)
		if err != nil {
			sentEventsErrors++
			glog.Errorf("failed to save events to ElasticSearch cluster: %s", err)
		} else {
			sentEvents++
		}
	}
	end := time.Now()
	glog.V(4).Infof("Exported %d data to ElasticSearch in %s", sentEvents, end.Sub(start))
}

func (sink *elasticSearchSink) Name() string {
	return "ElasticSearch Sink"
}

func (sink *elasticSearchSink) Stop() {
	// nothing needs to be done.
}

func (sink *elasticSearchSink) createIndex() error {
	if sink.client == nil {
		client, err := es_common.NewClient(sink.c)
		if err != nil {
			return err
		}
		sink.client = client
	}
	if sink.indexExists {
		return nil
	}
	// Use the IndexExists service to check if a specified index exists.
	exists, err := sink.client.IndexExists(sink.c.EventsIndex).Do()
	if err != nil {
		return err
	}
	if !exists {
		// Create a new index.
		createIndex, err := sink.client.CreateIndex(sink.c.EventsIndex).Do()
		if err != nil {
			return err
		}
		if !createIndex.Acknowledged {
			return fmt.Errorf("failed to create Index in ElasticSearch cluster: %s", err)
		}
	}
	sink.indexExists = true
	glog.Infof("Created index %q on ElasticSearch nodes at %s",
		sink.c.EventsIndex, strings.Join(sink.c.Nodes, ","))
	return nil
}

// Returns a thread-safe implementation of core.EventSink for InfluxDB.
func new(c es_common.ElasticSearchConfig) core.EventSink {
	client, err := es_common.NewClient(c)
	if err != nil {
		glog.Errorf("issues while creating an ElasticSearch sink: %v, will retry on use", err)
	}
	return &elasticSearchSink{
		client:       client, // can be nil
		c:            c,
		sendDataFunc: es_common.Write,
		indexExists:  false,
	}
}

func CreateElasticSearchSink(uri *url.URL) (core.EventSink, error) {
	config, err := es_common.BuildConfig(uri)
	if err != nil {
		return nil, err
	}
	config.EventsIndex = EventsIndex
	sink := new(*config)
	glog.Infof("created ElasticSearch sink with options: nodes:%s user:%s index:%s",
		strings.Join(config.Nodes, ","), config.User, config.EventsIndex)

	return sink, nil
}
