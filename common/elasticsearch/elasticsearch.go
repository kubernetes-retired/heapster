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

	"github.com/golang/glog"
	"github.com/olivere/elastic"
	"github.com/pborman/uuid"
)

const (
	ESIndex = "heapster"
)

// SaveDataFunc is a pluggable function to enforce limits on the object
type SaveDataFunc func(esClient *elastic.Client, indexName string, typeName string, sinkData interface{}) error

type ElasticSearchConfig struct {
	EsClient     *elastic.Client
	Index        string
	NeedAuthen   bool
	EsUserName   string
	EsUserSecret string
	EsNodes      []string
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

func CreateElasticSearchConfig(uri *url.URL) (*ElasticSearchConfig, error) {

	var esConfig ElasticSearchConfig
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parser url's query string: %s", err)
	}

	// set the index for es,the default value is "heapster"
	esConfig.Index = ESIndex
	if len(opts["index"]) > 0 {
		esConfig.Index = opts["index"][0]
	}

	// If the ES cluster needs authentication, the username and secret
	// should be set in sink config.Else, set the Authenticate flag to false
	esConfig.NeedAuthen = false
	if len(opts["esUserName"]) > 0 && len(opts["esUserSecret"]) > 0 {
		esConfig.EsUserName = opts["esUserName"][0]
		esConfig.NeedAuthen = true
	}

	// set the URL endpoints of the ES's nodes. Notice that
	// when sniffing is enabled, these URLs are used to initially sniff the
	// cluster on startup.
	if len(opts["nodes"]) < 1 {
		return nil, fmt.Errorf("There is no node assigned for connecting ES cluster")
	}
	esConfig.EsNodes = append(esConfig.EsNodes, opts["nodes"]...)
	glog.V(2).Infof("configing elasticsearch sink with ES's nodes - %v", esConfig.EsNodes)

	var client *(elastic.Client)
	if esConfig.NeedAuthen == false {
		client, err = elastic.NewClient(elastic.SetURL(esConfig.EsNodes...))
	} else {
		client, err = elastic.NewClient(elastic.SetBasicAuth(esConfig.EsUserName, esConfig.EsUserSecret), elastic.SetURL(esConfig.EsNodes...))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create ElasticSearch client: %v", err)
	}
	esConfig.EsClient = client
	glog.V(2).Infof("elasticsearch sink configure successfully")
	return &esConfig, nil
}
