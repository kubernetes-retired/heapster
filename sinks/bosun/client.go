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

package bosun

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"bosun.org/metadata"
	"bosun.org/opentsdb"

	"github.com/golang/glog"
)

type bosunClient interface {
	sendMetadata(ms []metadata.Metasend) error
	sendBatch(batch []json.RawMessage) error
}

type bosunClientImpl struct {
	metricsUrl *url.URL
	metaUrl    *url.URL
	client     *http.Client
}

func (bs *bosunClientImpl) sendMetadata(ms []metadata.Metasend) error {
	b, err := json.Marshal(&ms)
	if err != nil {
		return err
	}
	resp, err := http.Post(bs.metaUrl.String(), "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("Bosun Sink: bad metadata return:", resp.Status)
	}
	return nil
}

type sendError struct {
	success int64 `json:"success,omitempty"`
	failed  int64 `json:"failed,omitempty"`
	errors  []struct {
		datapoint opentsdb.DataPoint `json:"datapoint,omitempty"`
		error     string             `json:"error,omitempty"`
	} `json:"errors,omitempty"`
}

func (bs *bosunClientImpl) sendBatch(batch []json.RawMessage) error {
	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)
	if err := json.NewEncoder(g).Encode(batch); err != nil {
		return err
	}
	if err := g.Close(); err != nil {
		return err
	}
	q := bs.metricsUrl.Query()
	q.Set("details", "")
	bs.metricsUrl.RawQuery = q.Encode()
	req, err := http.NewRequest("POST", bs.metricsUrl.String(), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := bs.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}
	// Some problem with connecting to the server; retry later.
	if err != nil || resp.StatusCode != http.StatusNoContent {
		if err != nil {
			return err
		} else if resp.StatusCode != http.StatusNoContent {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to parse response body: %v", err)
			}
			se := sendError{}
			if err := json.Unmarshal(body, &se); err != nil {
				return fmt.Errorf("failed to parse response body: %v", err)
			}
			if se.failed == 0 {
				return nil
			}
			for _, error := range se.errors {
				glog.Errorf("failed to push datapoint: %v\nError: %v", error.datapoint, error.error)
			}
			return fmt.Errorf("Bosun Sink: Failed to push %d data points", se.failed)
		}
	}
	return nil
}
