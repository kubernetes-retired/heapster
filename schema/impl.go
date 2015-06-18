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

package schema

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
)

var lock sync.RWMutex

func NewCluster() Cluster {
	return newRealCluster()
}

func newRealCluster() *realCluster {
	cinfo := ClusterInfo{
		InfoType:   newInfoType(nil, nil),
		Namespaces: make(map[string]*NamespaceInfo),
		Nodes:      make(map[string]*NodeInfo),
	}
	cluster := &realCluster{
		timestamp:   time.Time{},
		ClusterInfo: cinfo,
	}
	return cluster
}

func (rc *realCluster) GetAllClusterData() (*ClusterInfo, time.Time, error) {
	// Returns the entire ClusterInfo object
	lock.RLock()
	defer lock.RUnlock()
	return &rc.ClusterInfo, rc.timestamp, nil
}

func (rc *realCluster) GetAllNodeData(hostname string) (*NodeInfo, time.Time, error) {
	// Returns the corresponding NodeInfo object
	var zeroTime time.Time

	lock.RLock()
	defer lock.RUnlock()

	res, ok := rc.Nodes[hostname]
	if !ok {
		return nil, zeroTime, fmt.Errorf("Unable to find node with hostname: %s", hostname)
	}

	return res, rc.timestamp, nil
}

func (rc *realCluster) GetAllPodData(namespace string, pod_name string) (*PodInfo, time.Time, error) {
	// Returns the corresponding NodeInfo object
	var zeroTime time.Time

	lock.RLock()
	defer lock.RUnlock()

	if len(rc.Namespaces) == 0 {
		return nil, zeroTime, fmt.Errorf("Unable to find pod: no namespaces in cluster")
	}

	ns, ok := rc.Namespaces[namespace]
	if !ok {
		return nil, zeroTime, fmt.Errorf("Unable to find namespace with name: %s", namespace)
	}

	pod, ok := ns.Pods[pod_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("Unable to find pod with name: %s", pod_name)
	}

	return pod, rc.timestamp, nil
}

func (rc *realCluster) Update(c *cache.Cache) error {
	return nil
}
