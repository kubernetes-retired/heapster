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

package repr

import (
	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	"github.com/golang/glog"
	"time"
)

func NewCluster() Cluster {
	var zero time.Time
	namespaces := make(map[string]*NamespaceInfo)
	nodes := make(map[string]*NodeInfo)
	cinfo := ClusterInfo{newInfoType(nil, nil), namespaces, nodes}
	cluster := &realCluster{zero, cinfo}
	return cluster
}

func (rc *realCluster) GetAllClusterData() (*ClusterInfo, time.Time, error) {
	/* Returns the entire ClusterInfo object */
	return &rc.ClusterInfo, time.Now(), nil
}

func (rc *realCluster) Update(c cache.Cache) error {
	/*
	 *	Get new data from cache and update data structure
	 */
	var zero time.Time

	// Invoke cache interface since the last timestamp
	nodes := c.GetNodes(rc.timestamp, zero)
	latest_time := rc.timestamp

	for _, node := range nodes {
		timestamp := rc.updateNode(node)
		latest_time = maxTimestamp(latest_time, timestamp)
	}

	pods := c.GetPods(rc.timestamp, zero)
	for _, pod := range pods {
		rc.updatePod(pod)
	}

	free_containers := c.GetFreeContainers(rc.timestamp, zero)
	for _, container := range free_containers {
		rc.updateContainer(container)
	}

	// Update the Cluster timestamp to the latest time found in the Update sequence
	rc.lock.Lock()
	defer rc.lock.Unlock()
	rc.timestamp = latest_time

	return nil
}

func (rc *realCluster) updateNode(node_container *cache.ContainerElement) time.Time {
	/*	Inserts or updates Node information from the
	 *	"machine"-tagged containers
	 */
	container_name := node_container.Name
	if container_name != "machine" {
		glog.Fatalf("Received node-level container with unexpected name")
	}

	node_name := node_container.Hostname

	if _, ok := rc.Nodes[node_name]; !ok {
		// Node name does not exist, create a new NodeInfo object
		rc.addNode(node_name)
	}

	// Update NodeInfo's Metrics and Labels - return latest metric timestamp
	return updateInfoType(&rc.Nodes[node_name].InfoType, node_container)
}

func (rc *realCluster) addNode(hostname string) *NodeInfo {
	/*	Creates a new NodeInfo object for the provided hostname
	 */

	// Allocate all new structures
	pods := make(map[string]*PodInfo)
	free_conts := make(map[string]*ContainerInfo)
	new_node := NodeInfo{newInfoType(nil, nil), pods, free_conts}

	// Obtain Cluster Lock and point to new_node
	rc.lock.Lock()
	defer rc.lock.Unlock()
	rc.Nodes[hostname] = &new_node

	return &new_node
}

func (rc *realCluster) updatePod(pod *cache.PodElement) {
	/*	Inserts or updates Pod information from
	 *	pod-labelled containers
	 */
}

func (rc *realCluster) updateContainer(container *cache.ContainerElement) {
	/*	Inserts or updates Container information from
	 *	a single container element
	 */
}
