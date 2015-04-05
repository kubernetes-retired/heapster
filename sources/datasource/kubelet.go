// Copyright 2014 Google Inc. All Rights Reserved.
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

// This file implements a cadvisor datasource, that collects metrics from an instance
// of cadvisor runing on a specific host.

package datasource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info/v1"
)

type kubeletSource struct{}

func (self *kubeletSource) postRequestAndGetValue(client *http.Client, req *http.Request, value interface{}) error {
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body - %v", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed - %q, response: %q", response.Status, string(body))
	}
	err = json.Unmarshal(body, value)
	if err != nil {
		return fmt.Errorf("failed to parse output. Response: %q. Error: %v", string(body), err)
	}
	return nil
}

func (self *kubeletSource) parseStat(containerInfo *cadvisor.ContainerInfo, resolution time.Duration) *api.Container {
	if len(containerInfo.Stats) == 0 {
		return nil
	}
	container := &api.Container{
		Name:  containerInfo.Name,
		Spec:  containerInfo.Spec,
		Stats: sampleContainerStats(containerInfo.Stats, resolution),
	}
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}

	return container
}

func (self *kubeletSource) getContainer(url string, start, end time.Time, resolution time.Duration) (*api.Container, error) {
	// TODO: Get rid of 'NumStats' once cadvisor supports time range queries without specifying that.
	body, err := json.Marshal(cadvisor.ContainerInfoRequest{Start: start, End: end, NumStats: int(time.Minute / time.Second)})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var containerInfo cadvisor.ContainerInfo
	err = self.postRequestAndGetValue(http.DefaultClient, req, &containerInfo)
	if err != nil {
		glog.Errorf("failed to get stats from kubelet url: %s - %s\n", url, err)
		return nil, err
	}
	glog.V(4).Infof("url: %q, body: %q, data: %+v", url, string(body), containerInfo)
	return self.parseStat(&containerInfo, resolution), nil
}

func (self *kubeletSource) GetContainer(host Host, start, end time.Time, resolution time.Duration) (container *api.Container, err error) {
	url := fmt.Sprintf("https://%s:%s/%s", host.IP, host.Port, host.Resource)
	glog.V(3).Infof("about to query kubelet using url: %q", url)

	return self.getContainer(url, start, end, resolution)
}
