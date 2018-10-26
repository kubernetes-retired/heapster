// Copyright 2017 Google Inc. All Rights Reserved.
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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
)

const maxBatchSize = 100

type config struct {
	APIHost  string
	Dataset  string
	WriteKey string
}

func BuildConfig(uri *url.URL) (*config, error) {
	opts := uri.Query()

	config := &config{
		WriteKey: os.Getenv("HONEYCOMB_WRITEKEY"),
		APIHost:  "https://api.honeycomb.io/",
		Dataset:  "heapster",
	}

	if len(opts["writekey"]) >= 1 {
		config.WriteKey = opts["writekey"][0]
	}

	if len(opts["apihost"]) >= 1 {
		config.APIHost = opts["apihost"][0]
	}

	if len(opts["dataset"]) >= 1 {
		config.Dataset = opts["dataset"][0]
	}

	if config.WriteKey == "" {
		return nil, errors.New("Failed to find honeycomb API write key")
	}

	return config, nil
}

type Client interface {
	SendBatch(batch Batch) error
}

type HoneycombClient struct {
	config     config
	httpClient http.Client
}

func NewClient(uri *url.URL) (*HoneycombClient, error) {
	config, err := BuildConfig(uri)
	if err != nil {
		return nil, err
	}
	return &HoneycombClient{config: *config}, nil
}

type BatchPoint struct {
	Data      interface{}
	Timestamp time.Time
}

type Batch []*BatchPoint

func (c *HoneycombClient) sendBatch(batch Batch) error {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(batch)
	if err != nil {
		return err
	}
	err = c.makeRequest(buf)
	if err != nil {
		return err
	}
	return nil
}

// SendBatch splits the top-level batch into sub-batches if needed.  Otherwise,
// requests that are too large will be rejected by the Honeycomb API.
func (c *HoneycombClient) SendBatch(batch Batch) error {
	if len(batch) == 0 {
		// Nothing to send
		return nil
	}

	errs := []string{}
	for i := 0; i < len(batch); i += maxBatchSize {
		offset := i + maxBatchSize
		if offset > len(batch) {
			offset = len(batch)
		}
		if err := c.sendBatch(batch[i:offset]); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func (c *HoneycombClient) makeRequest(body io.Reader) error {
	url, err := url.Parse(c.config.APIHost)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "/1/batch", c.config.Dataset)
	req, err := http.NewRequest("POST", url.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("X-Honeycomb-Team", c.config.WriteKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		glog.Warningf("Failed to send event: %v", err)
		return err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
	return nil
}
