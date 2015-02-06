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

package nodes

import (
	"fmt"
	"net"
	"sync"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/cache"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/golang/glog"
)

type kubeNodes struct {
	client *client.Client
	// a means to list all minions
	minionLister *cache.StoreToNodeLister
	goodNodes    []string       // guarded by stateLock
	nodeErrors   map[string]int // guarded by stateLock
	stateLock    sync.RWMutex
}

func (self *kubeNodes) recordNodeError(name string) {
	self.stateLock.Lock()
	defer self.stateLock.Unlock()

	self.nodeErrors[name]++
}

func (self *kubeNodes) recordGoodNodes(nodes []string) {
	self.stateLock.Lock()
	defer self.stateLock.Unlock()

	self.goodNodes = nodes
}

func parseSelectorOrDie(s string) labels.Selector {
	selector, err := labels.ParseSelector(s)
	if err != nil {
		panic(err)
	}
	return selector
}

func (self *kubeNodes) createMinionLW() *cache.ListWatch {
	return &cache.ListWatch{
		Client:        self.client,
		FieldSelector: parseSelectorOrDie(""),
		Resource:      "minions",
	}
}

func (self *kubeNodes) List() (*NodeList, error) {
	nodeList := &NodeList{Items: map[Node]Empty{}}
	allNodes, err := self.minionLister.List()
	if err != nil {
		glog.Errorf("failed to list minions via watch interface - %v", err)
		return nil, fmt.Errorf("failed to list minions via watch interface - %v", err)
	}
	glog.V(3).Infof("all kube nodes: %+v", allNodes)

	goodNodes := []string{}
	for _, node := range allNodes.Items {
		// TODO(vishh): Consider dropping nodes that are not healthy as indicated in node.Status.Phase.
		if node.Status.HostIP != "" {
			nodeList.Items[Node{node.Name, node.Status.HostIP}] = Empty{}
			goodNodes = append(goodNodes, node.Name)
			continue
		}
		// TODO(vishh): Remove this logic once Status.HostIP is reliable.
		addrs, err := net.LookupIP(node.Name)
		if err == nil {
			nodeList.Items[Node{node.Name, addrs[0].String()}] = Empty{}
			goodNodes = append(goodNodes, node.Name)
		} else {
			glog.Errorf("Skipping host %s since looking up its IP failed - %s", node.Name, err)
			self.recordNodeError(node.Name)
		}
	}
	self.recordGoodNodes(goodNodes)
	glog.V(2).Infof("kube nodes found: %+v", nodeList)
	return nodeList, nil
}

func (self *kubeNodes) getState() string {
	self.stateLock.RLock()
	defer self.stateLock.RUnlock()

	state := "\tHealthy Nodes:\n"
	for _, node := range self.goodNodes {
		state += fmt.Sprintf("\t\t%s\n", node)
	}
	if len(self.nodeErrors) != 0 {
		state += fmt.Sprintf("\tNode Errors: %+v\n", self.nodeErrors)
	} else {
		state += "\tNo node errors\n"
	}
	return state
}

func (self *kubeNodes) DebugInfo() string {
	desc := "Node watcher: Kubernetes\n"
	desc += self.getState()
	desc += "\n"

	return desc
}

func NewKubeNodes(client *client.Client) (NodesApi, error) {
	if client == nil {
		return nil, fmt.Errorf("client is nil")
	}

	kubeNodes := &kubeNodes{
		client:       client,
		minionLister: &cache.StoreToNodeLister{cache.NewStore(cache.MetaNamespaceKeyFunc)},
		nodeErrors:   make(map[string]int),
	}
	cache.NewReflector(kubeNodes.createMinionLW(), &api.Node{}, kubeNodes.minionLister.Store).Run()

	return kubeNodes, nil
}
