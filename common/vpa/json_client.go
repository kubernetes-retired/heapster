/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vpa

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type httpJSONClient struct {
	url string
}

type JSONClient interface {
	SendJSON(object interface{}) ([]byte, error)
}

func CreateRecommenderClient(url string) JSONClient {
	return &httpJSONClient{url: url}
}

func (c *httpJSONClient) SendJSON(object interface{}) ([]byte, error) {
	data, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	response, err := c.sendData(data, "application/json")
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *httpJSONClient) sendData(data []byte, dataType string) ([]byte, error) {
	resp, err := http.Post(c.url, dataType, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}
