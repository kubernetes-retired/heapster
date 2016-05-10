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
	"strconv"
	"strings"

	"github.com/pborman/uuid"
	elastic "gopkg.in/olivere/elastic.v3"
)

type ElasticSearchClient interface {
	Index() *elastic.IndexService
	IndexExists(...string) *elastic.IndicesExistsService
	CreateIndex(string) *elastic.IndicesCreateService
}

type ElasticSearchConfig struct {
	User         string
	Password     string
	Secure       bool
	Nodes        []string
	EventsIndex  string
	MetricsIndex string
}

func NewClient(c ElasticSearchConfig) (ElasticSearchClient, error) {
	var err error
	var client ElasticSearchClient
	setSniff := true
	if len(c.Nodes) == 1 {
		setSniff = false
	}
	if c.Secure == false {
		client, err = elastic.NewClient(
			elastic.SetSniff(setSniff),
			elastic.SetURL(strings.Join(c.Nodes, ",")))
	} else {
		client, err = elastic.NewClient(
			elastic.SetSniff(setSniff),
			elastic.SetBasicAuth(c.User, c.Password), elastic.SetURL(c.Nodes...))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect any node of ElasticSearch cluster")
	}
	return client, nil
}

//Write save metrics and events to ElasticSearch by using ElasticSearch client
func Write(esClient ElasticSearchClient, indexName string, typeName string, sinkData interface{}) error {
	if indexName == "" || typeName == "" || sinkData == nil {
		return nil
	}
	// Send data
	indexID := uuid.NewUUID()
	_, err := esClient.Index().
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

func BuildConfig(uri *url.URL) (*ElasticSearchConfig, error) {
	config := ElasticSearchConfig{
		Secure: false,
	}

	if len(uri.Host) > 0 {
		for _, node := range strings.Split(uri.Host, ",") {
			config.Nodes = append(config.Nodes, uri.Scheme+"://"+node)
		}
	}
	opts := uri.Query()
	if len(opts["user"]) >= 1 {
		config.User = opts["user"][0]
	}
	// TODO: use more secure way to pass the password.
	if len(opts["pw"]) >= 1 {
		config.Password = opts["pw"][0]
	}
	if len(opts["secure"]) >= 1 {
		val, err := strconv.ParseBool(opts["secure"][0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse `secure` flag - %v", err)
		}
		config.Secure = val
	}

	return &config, nil
}
