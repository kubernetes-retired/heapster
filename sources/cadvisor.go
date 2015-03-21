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

// This file is dedicated to heapster running outside of kubernetes. Heapster
// will poll a file to get the hosts that it needs to monitor and will collect
// stats from cadvisor running on those hosts.

package sources

import (
	"flag"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/sources/datasource"
	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	"github.com/golang/glog"
)

var (
	argCadvisorPort = flag.Int("cadvisor_port", 8080, "The port on which cadvisor binds to on all nodes.")
	argCoreOSMode   = flag.Bool("coreos", false, "When true, heapster looks will connect with fleet servers to get the list of nodes to monitor. It is expected that cadvisor will be running on all the nodes at the port specified using flag '--cadvisor_port'. Use flag '--fleet_endpoints' to manage fleet endpoints to watch.")
)

type cadvisorSource struct {
	cadvisorPort string
	cadvisorApi  datasource.Cadvisor
	pollDuration time.Duration
	nodesApi     nodes.NodesApi
}

func (self *cadvisorSource) GetInfo(start, end time.Time, resolution time.Duration) (api.AggregateData, error) {
	var (
		lock sync.Mutex
		wg   sync.WaitGroup
	)
	nodeList, err := self.nodesApi.List()
	if err != nil {
		return api.AggregateData{}, err
	}

	result := api.AggregateData{}
	for hostname, info := range nodeList.Items {
		wg.Add(1)
		go func(hostname string, info nodes.Info) {
			defer wg.Done()
			host := datasource.Host{
				IP:   info.InternalIP,
				Port: self.cadvisorPort,
			}
			rawSubcontainers, node, err := self.cadvisorApi.GetAllContainers(host, start, end, resolution)
			if err != nil {
				glog.Error(err)
				return
			}
			subcontainers := []api.Container{}
			for _, cont := range rawSubcontainers {
				if cont != nil {
					cont.Hostname = hostname
					subcontainers = append(subcontainers, *cont)
				}
			}
			lock.Lock()
			defer lock.Unlock()
			result.Containers = append(result.Containers, subcontainers...)
			if node != nil {
				node.Hostname = hostname
				result.Machine = append(result.Machine, *node)
			}
		}(string(hostname), info)
	}
	wg.Wait()

	return result, nil
}

func (self *cadvisorSource) DebugInfo() string {
	desc := "Source type: Cadvisor\n"
	// TODO(rjnagal): Cache config?
	nodeList, err := self.nodesApi.List()
	if err != nil {
		desc += fmt.Sprintf("\tFailed to read host config: %s", err)
	}
	desc += fmt.Sprintf("\tNodeList: %+v\n", *nodeList)
	desc += fmt.Sprintf("\t%s\n", self.nodesApi.DebugInfo())
	desc += "\n"
	return desc
}

func newExternalCadvisorSource(pollDuration time.Duration) (Source, error) {
	if *argCadvisorPort <= 0 {
		return nil, fmt.Errorf("invalid cadvisor port - %d", *argCadvisorPort)
	}
	nodesApi, err := nodes.NewExternalNodes()
	if err != nil {
		return nil, err
	}
	return &cadvisorSource{
		pollDuration: pollDuration,
		cadvisorApi:  datasource.NewCadvisor(),
		nodesApi:     nodesApi,
		cadvisorPort: strconv.Itoa(*argCadvisorPort),
	}, nil
}

func newCoreOSCadvisorSource(pollDuration time.Duration) (Source, error) {
	if *argCadvisorPort <= 0 {
		return nil, fmt.Errorf("invalid cadvisor port - %d", *argCadvisorPort)
	}
	nodesApi, err := nodes.NewCoreOSNodes()
	if err != nil {
		return nil, err
	}
	return &cadvisorSource{
		pollDuration: pollDuration,
		cadvisorApi:  datasource.NewCadvisor(),
		nodesApi:     nodesApi,
		cadvisorPort: strconv.Itoa(*argCadvisorPort),
	}, nil
}
