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

	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info"
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
		return err
	}
	err = json.Unmarshal(body, value)
	if err != nil {
		return fmt.Errorf("Got '%s': %v", string(body), err)
	}
	return nil
}

func (self *kubeletSource) parseStat(containerInfo *cadvisor.ContainerInfo) *api.Container {
	if len(containerInfo.Stats) == 0 {
		return nil
	}
	container := &api.Container{
		Name:  containerInfo.Name,
		Spec:  containerInfo.Spec,
		Stats: containerInfo.Stats,
	}
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}

	return container
}

func (self *kubeletSource) getContainer(url string, numStats int) (*api.Container, error) {
	body, err := json.Marshal(cadvisor.ContainerInfoRequest{NumStats: numStats})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
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

	return self.parseStat(&containerInfo), nil
}

func (self *kubeletSource) GetContainer(host Host, numStats int) (container *api.Container, err error) {
	url := fmt.Sprintf("http://%s:%s/%s", host.IP, host.Port, host.Resource)
	glog.V(2).Infof("about to query kubelet using url: %q", url)

	return self.getContainer(url, numStats)
}
