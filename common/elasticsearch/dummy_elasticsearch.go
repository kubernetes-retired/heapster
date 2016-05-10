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
	elastic "gopkg.in/olivere/elastic.v3"
)

type PointSavedToElasticSearch struct {
	indexName string
	typeName  string
	sinkData  interface{}
}

type FakeElasticSearchClient struct {
	Pnts []PointSavedToElasticSearch
}

func NewFakeElasticSearchClient() *FakeElasticSearchClient {
	return &FakeElasticSearchClient{[]PointSavedToElasticSearch{}}
}

func (client *FakeElasticSearchClient) Write(esClient ElasticSearchClient, indexName string, typeName string, sinkData interface{}) error {
	pnt := PointSavedToElasticSearch{
		indexName: indexName,
		typeName:  typeName,
		sinkData:  sinkData,
	}
	client.Pnts = append(client.Pnts, pnt)
	return nil
}

func (client *FakeElasticSearchClient) Index() *elastic.IndexService {
	return nil
}

func (client *FakeElasticSearchClient) CreateIndex(string) *elastic.IndicesCreateService {
	return nil
}

func (client *FakeElasticSearchClient) IndexExists(...string) *elastic.IndicesExistsService {
	return nil
}

var Client = NewFakeElasticSearchClient()

var Config = ElasticSearchConfig{
	Secure:       false,
	Nodes:        []string{"127.0.0.1:9200"},
	EventsIndex:  "heapster-events",
	MetricsIndex: "heapster-metrics",
}
