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

	"github.com/golang/glog"
	"github.com/olivere/elastic"
	"github.com/pborman/uuid"
	"k8s.io/heapster/metrics/core"
)

const (
	timeSeriesIndex = "heapster-metrics"
	typeName        = "k8s-heapster"
)

// LimitFunc is a pluggable function to enforce limits on the object
type SaveDataFunc func(esClient *elastic.Client, indexName string, typeName string, sinkData interface{}) error

type elasticSearchSink struct {
	esClient        *elastic.Client
	saveDataFunc    SaveDataFunc
	timeSeriesIndex string
	needAuthen      bool
	esUserName      string
	esUserSecret    string
	esNodes         []string
	sync.RWMutex
}

type EsSinkPoint struct {
	MetricsName      string
	MetricsValue     interface{}
	MetricsTimestamp time.Time
	MetricsTags      map[string]string
}

func (sink *elasticSearchSink) ExportData(dataBatch *core.DataBatch) {
	sink.Lock()
	defer sink.Unlock()

	for _, metricSet := range dataBatch.MetricSets {
		for metricName, metricValue := range metricSet.MetricValues {
			point := EsSinkPoint{
				MetricsName: metricName,
				MetricsTags: metricSet.Labels,
				MetricsValue: map[string]interface{}{
					"value": metricValue.GetValue(),
				},
				MetricsTimestamp: dataBatch.Timestamp.UTC(),
			}
			sink.saveDataFunc(sink.esClient, sink.timeSeriesIndex, typeName, point)
		}
		for _, metric := range metricSet.LabeledMetrics {
			labels := make(map[string]string)
			for k, v := range metricSet.Labels {
				labels[k] = v
			}
			for k, v := range metric.Labels {
				labels[k] = v
			}
			point := EsSinkPoint{
				MetricsName: metric.Name,
				MetricsTags: labels,
				MetricsValue: map[string]interface{}{
					"value": metric.GetValue(),
				},
				MetricsTimestamp: dataBatch.Timestamp.UTC(),
			}
			sink.saveDataFunc(sink.esClient, sink.timeSeriesIndex, typeName, point)
		}
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

func NewElasticSearchSink(uri *url.URL) (core.DataSink, error) {

	var esSink elasticSearchSink
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parser url's query string: %s", err)
	}

	//set the index for timeSeries,the default value is "timeSeriesIndex"
	esSink.timeSeriesIndex = timeSeriesIndex
	if len(opts["timeseriesIndex"]) > 0 {
		esSink.timeSeriesIndex = opts["timeseriesIndex"][0]
	}

	//If the ES cluster needs authentication, the username and secret
	//should be set in sink config.Else, set the Authenticate flag to false
	esSink.needAuthen = false
	if len(opts["esUserName"]) > 0 && len(opts["esUserSecret"]) > 0 {
		esSink.timeSeriesIndex = opts["esUserName"][0]
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
