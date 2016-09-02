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
	"time"

	"github.com/golang/glog"
	"github.com/pborman/uuid"

	"gopkg.in/olivere/elastic.v3"
	"os"
)

const (
	ESIndex = "heapster"
)

// SaveDataFunc is a pluggable function to enforce limits on the object
type SaveDataFunc func(esClient *elastic.Client, indexName string, typeName string, sinkData interface{}) error

type ElasticSearchConfig struct {
	EsClient *elastic.Client
	Index    string
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
		createIndex, err := esClient.CreateIndex(indexName).BodyString(mapping).Do()
		if err != nil {
			return err
		}
		if !createIndex.Acknowledged {
			return fmt.Errorf("Failed to create Index in ES cluster: %s", err)
		}
	}
	indexID := uuid.NewUUID()
	_, err = esClient.Index().
		Index(indexName).
		Type(typeName).
		Id(indexID.String()).
		BodyJson(sinkData).
		Do()
	if err != nil {
		return err
	}
	return nil
}

// CreateElasticSearchConfig creates an ElasticSearch configuration struct
// which contains an ElasticSearch client for later use
func CreateElasticSearchConfig(uri *url.URL) (*ElasticSearchConfig, error) {

	var esConfig ElasticSearchConfig
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("Failed to parser url's query string: %s", err)
	}

	// set the index for es,the default value is "heapster"
	esConfig.Index = ESIndex
	if len(opts["index"]) > 0 {
		esConfig.Index = opts["index"][0]
	}

	// Set the URL endpoints of the ES's nodes. Notice that when sniffing is
	// enabled, these URLs are used to initially sniff the cluster on startup.
	var startupFns []elastic.ClientOptionFunc
	if len(opts["nodes"]) > 0 {
		startupFns = append(startupFns, elastic.SetURL(opts["nodes"]...))
	} else if uri.Opaque != "" {
		startupFns = append(startupFns, elastic.SetURL(uri.Opaque))
	} else {
		return nil, fmt.Errorf("There is no node assigned for connecting ES cluster")
	}

	// If the ES cluster needs authentication, the username and secret
	// should be set in sink config.Else, set the Authenticate flag to false
	if len(opts["esUserName"]) > 0 && len(opts["esUserSecret"]) > 0 {
		startupFns = append(startupFns, elastic.SetBasicAuth(opts["esUserName"][0], opts["esUserSecret"][0]))
	}

	if len(opts["maxRetries"]) > 0 {
		maxRetries, err := strconv.Atoi(opts["maxRetries"][0])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse URL's maxRetries value into an int")
		}
		startupFns = append(startupFns, elastic.SetMaxRetries(maxRetries))
	}

	if len(opts["healthCheck"]) > 0 {
		healthCheck, err := strconv.ParseBool(opts["healthCheck"][0])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse URL's healthCheck value into a bool")
		}
		startupFns = append(startupFns, elastic.SetHealthcheck(healthCheck))
	}

	if len(opts["startupHealthcheckTimeout"]) > 0 {
		timeout, err := time.ParseDuration(opts["startupHealthcheckTimeout"][0] + "s")
		if err != nil {
			return nil, fmt.Errorf("Failed to parse URL's startupHealthcheckTimeout: %s", err.Error())
		}
		startupFns = append(startupFns, elastic.SetHealthcheckTimeoutStartup(timeout))
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_ACCESS_KEY") != "" ||
		os.Getenv("AWS_SECRET_ACCESS_KEY") != "" || os.Getenv("AWS_SECRET_KEY") != "" {
		glog.Info("Configuring with AWS credentials..")

		awsClient, err := createAWSClient()
		if err != nil {
			return nil, err
		}

		startupFns = append(startupFns, elastic.SetHttpClient(awsClient), elastic.SetSniff(false))
	} else {
		if len(opts["sniff"]) > 0 {
			sniff, err := strconv.ParseBool(opts["sniff"][0])
			if err != nil {
				return nil, fmt.Errorf("Failed to parse URL's sniff value into a bool")
			}
			startupFns = append(startupFns, elastic.SetSniff(sniff))
		}
	}

	esConfig.EsClient, err = elastic.NewClient(startupFns...)
	if err != nil {
		return nil, fmt.Errorf("Failed to create ElasticSearch client: %v", err)
	}

	glog.V(2).Infof("ElasticSearch sink configure successfully")

	return &esConfig, nil
}
