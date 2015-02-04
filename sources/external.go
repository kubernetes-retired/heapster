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
	"flag"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	"github.com/golang/glog"
)

var argCadvisorPort = flag.Int("cadvisor_port", 8080, "The port on which cadvisor binds to on all nodes.")

type externalSource struct {
	cadvisor     *cadvisorSource
	pollDuration time.Duration
	nodesApi nodes.NodesApi
}

func (self *externalSource) GetInfo() (ContainerData, error) {
	nodeList, err := self.nodesApi.List()
	if err != nil {
		return ContainerData{}, err
	}

	containers, nodes, err := self.cadvisor.fetchData(nodeList.Items)
	if err != nil {
		glog.Error(err)
		return ContainerData{}, nil
	}

	return ContainerData{
		Containers: containers,
		Machine:    nodes,
	}, nil
}

func newExternalSource(pollDuration time.Duration) (Source, error) {
	cadvisorSource, err := newCadvisorSource(pollDuration, *argCadvisorPort)
	if err != nil {
		return nil, err
	}
	nodesApi, err := nodes.NewExternalNodes()
	if err != nil {
		return nil, err
	}
	return &externalSource{
		cadvisor: cadvisorSource,
		nodesApi: nodesApi,
	}, nil
}

func (self *externalSource) GetConfig() string {
	desc := "Source type: External\n"
	// TODO(rjnagal): Cache config?
	nodeList, err := self.nodesApi.List()
	if err != nil {
		desc += fmt.Sprintf("\tFailed to read host config: %s", err)
	}
	desc += fmt.Sprintf("\tNodeList: %+v\n", *nodeList)
	desc += "\n"
	return desc
}
