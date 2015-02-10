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
	"fmt"

	"github.com/golang/glog"
	cadvisorClient "github.com/google/cadvisor/client"
	cadvisor "github.com/google/cadvisor/info"
)

type cadvisorSource struct{}

func (self *cadvisorSource) parseStat(containerInfo *cadvisor.ContainerInfo) *Container {
	container := &Container{
		Name:  containerInfo.Name,
		Spec:  containerInfo.Spec,
		Stats: containerInfo.Stats,
	}
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}

	return container
}

func (self *cadvisorSource) getAllContainers(client *cadvisorClient.Client, numStats int) (subcontainers []*Container, root *Container, err error) {
	allContainers, err := client.SubcontainersInfo("/",
		&cadvisor.ContainerInfoRequest{NumStats: numStats})
	if err != nil {

		return nil, nil, err
	}

	for _, containerInfo := range allContainers {
		container := self.parseStat(&containerInfo)
		if containerInfo.Name == "/" {
			root = container
		} else {
			subcontainers = append(subcontainers, container)
		}
	}

	return subcontainers, root, nil
}

func (self *cadvisorSource) GetAllContainers(host Host, numStats int) (subcontainers []*Container, root *Container, err error) {
	url := fmt.Sprintf("http://%s:%s/", host.IP, host.Port)
	client, err := cadvisorClient.NewClient(url)
	if err != nil {
		return
	}
	subcontainers, root, err = self.getAllContainers(client, numStats)
	if err != nil {
		glog.Errorf("failed to get stats from cadvisor %q - %v\n", url, err)
	}
	return
}
