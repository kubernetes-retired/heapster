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
	"errors"
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
	lock.RLock()
	defer lock.RUnlock()

	var res *NodeInfo
	var stamp time.Time
	var err error

	if val, ok := rc.Nodes[hostname]; ok {
		res = val
		stamp = rc.timestamp
		err = nil
	} else {
		res = nil
		err = errors.New("Unable to find node with hostname: " + hostname)
	}
	return res, stamp, err
}

func (rc *realCluster) GetAllPodData(namespace string, pod_name string) (*PodInfo, time.Time, error) {
	// Returns the corresponding NodeInfo object
	lock.RLock()
	defer lock.RUnlock()

	var res *PodInfo
	var stamp time.Time
	var err error

	if len(rc.Namespaces) == 0 {
		err = errors.New("Unable to find pod: no namespaces registered")
	} else {
		if ns, ok := rc.Namespaces[namespace]; ok {
			if val, ok := ns.Pods[pod_name]; ok {
				res = val
				stamp = rc.timestamp
				err = nil
			} else {
				err = errors.New("Unable to find pod with name: " + pod_name)
			}
		} else {
			err = errors.New("Unable to find namespace with name: " + namespace)
		}
	}
	return res, stamp, err
}

func (rc *realCluster) Update(c *cache.Cache) error {
	return nil
}
