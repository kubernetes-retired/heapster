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

package sources

import (
	"fmt"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	"github.com/golang/glog"
	cadvisorClient "github.com/google/cadvisor/client"
	cadvisor "github.com/google/cadvisor/info"
)

type cadvisorSource struct {
	pollDuration time.Duration
	port      string
	lastQuery time.Time
}

func (self *cadvisorSource) processStat(hostname string, containerInfo *cadvisor.ContainerInfo) RawContainer {
	container := Container{
		Name:  containerInfo.Name,
		Spec:  containerInfo.Spec,
		Stats: containerInfo.Stats,
	}
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}

	return RawContainer{hostname, container}
}

func (self *cadvisorSource) getAllCadvisorData(hostname, ip, port, container string) (containers []RawContainer, nodeInfo RawContainer, err error) {
	client, err := cadvisorClient.NewClient("http://" + ip + ":" + port)
	if err != nil {
		return
	}
	numStats := int(self.pollDuration / time.Second)
	if time.Since(self.lastQuery) > self.pollDuration {
		numStats = int(time.Since(self.lastQuery)/time.Second)
	}
	allContainers, err := client.SubcontainersInfo("/",
		&cadvisor.ContainerInfoRequest{NumStats: numStats})
	if err != nil {
		glog.Errorf("failed to get stats from cadvisor on host %s with ip %s - %s\n", hostname, ip, err)
		return
	}

	for _, containerInfo := range allContainers {
		rawContainer := self.processStat(hostname, &containerInfo)
		if containerInfo.Name == "/" {
			nodeInfo = rawContainer
		} else {
			containers = append(containers, rawContainer)
		}
	}

	return containers, nodeInfo, nil
}

func (self *cadvisorSource) fetchData(nodes []nodes.Node) (rawContainers []RawContainer, nodesInfo []RawContainer, err error) {
	for _, node := range nodes {
		containers, nodeInfo, err := self.getAllCadvisorData(node.Name, node.IP, self.port, "/")
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to get cAdvisor data from host %q: %v", node.Name, err)
		}
		rawContainers = append(rawContainers, containers...)
		nodesInfo = append(nodesInfo, nodeInfo)
	}
	self.lastQuery = time.Now()
	return rawContainers, nodesInfo, nil
}

func newCadvisorSource(pollDuration time.Duration, port int) (*cadvisorSource, error) {
	if port <= 0 {
		return nil, fmt.Errorf("cadvisor port invalid - %d", port)
	}
	return &cadvisorSource{
		pollDuration: pollDuration,
		port:      strconv.Itoa(port),
		lastQuery: time.Now(),
	}, nil
}
