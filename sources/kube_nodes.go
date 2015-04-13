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

package sources

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/sources/datasource"
	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	"github.com/golang/glog"
)

type kubeNodeMetrics struct {
	kubeletApi  datasource.Kubelet
	kubeletPort int
	nodesApi    nodes.NodesApi
}

func NewKubeNodeMetrics(kubeletPort int, kubeletApi datasource.Kubelet, nodesApi nodes.NodesApi) api.Source {
	return &kubeNodeMetrics{
		kubeletApi:  kubeletApi,
		kubeletPort: kubeletPort,
		nodesApi:    nodesApi,
	}
}

func (self *kubeNodeMetrics) updateStats(host nodes.Host, info nodes.Info, start, end time.Time, resolution time.Duration) (api.Container, error) {
	container, err := self.kubeletApi.GetContainer(datasource.Host{IP: info.InternalIP, Port: self.kubeletPort, Resource: "stats/"}, start, end, resolution)
	if err != nil {
		glog.V(1).Infof("Failed to get machine stats from kubelet for node %s", host)
		return api.Container{}, err
	}
	if container == nil {
		// no stats found.
		glog.V(1).Infof("no machine stats from kubelet on node %s", host)
		return api.Container{}, fmt.Errorf("no machine stats from kubelet on node %s", host)
	}
	container.Hostname = string(host)
	return *container, nil
}

func (self *kubeNodeMetrics) getNodesInfo(nodeList *nodes.NodeList, start, end time.Time, resolution time.Duration) ([]api.Container, error) {
	var (
		lock sync.Mutex
		wg   sync.WaitGroup
	)
	nodesInfo := []api.Container{}
	for host, info := range nodeList.Items {
		wg.Add(1)
		go func(host nodes.Host, info nodes.Info) {
			defer wg.Done()
			if container, err := self.updateStats(host, info, start, end, resolution); err == nil {
				lock.Lock()
				defer lock.Unlock()
				nodesInfo = append(nodesInfo, container)
			}
		}(host, info)
	}
	wg.Wait()

	return nodesInfo, nil
}

func (self *kubeNodeMetrics) GetInfo(start, end time.Time, resolution time.Duration) (api.AggregateData, error) {
	kubeNodes, err := self.nodesApi.List()
	if err != nil || len(kubeNodes.Items) == 0 {
		return api.AggregateData{}, err
	}
	glog.V(3).Info("Fetched list of nodes from the master")
	nodesInfo, err := self.getNodesInfo(kubeNodes, start, end, resolution)
	if err != nil {
		return api.AggregateData{}, err
	}

	return api.AggregateData{Machine: nodesInfo}, nil
}

func (self *kubeNodeMetrics) DebugInfo() string {
	desc := "Source type: Kube Node Metrics\n"
	desc += self.nodesApi.DebugInfo() + "\n"

	return desc
}
