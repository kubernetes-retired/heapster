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
	"github.com/GoogleCloudPlatform/heapster/api/schema/info"
	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	"github.com/golang/glog"
	"sync"
	"time"
)

func NewCluster() Cluster {
	return NewRealCluster()
}

func NewRealCluster() *realCluster {
	cinfo := info.ClusterInfo{
		InfoType:   newInfoType(nil, nil),
		Namespaces: make(map[string]*info.NamespaceInfo),
		Nodes:      make(map[string]*info.NodeInfo),
	}
	cluster := &realCluster{
		timestamp:   time.Time{},
		lock:        new(sync.RWMutex),
		ClusterInfo: cinfo,
	}
	return cluster
}

func (rc *realCluster) GetAllClusterData() (*info.ClusterInfo, time.Time, error) {
	// Returns the entire ClusterInfo object
	rc.lock.RLock()
	defer rc.lock.RUnlock()
	return &rc.ClusterInfo, rc.timestamp, nil
}

func (rc *realCluster) GetAllNodeData(hostname string) (*info.NodeInfo, time.Time, error) {
	// Returns the corresponding NodeInfo object
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	var res *info.NodeInfo
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

func (rc *realCluster) GetAllPodData(namespace string, pod_name string) (*info.PodInfo, time.Time, error) {
	// Returns the corresponding NodeInfo object
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	var res *info.PodInfo
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
	// Gets new data from cache and updates data structure
	var zero time.Time

	glog.V(2).Infoln("Schema Update operation started")

	actual_cache := *c // Dereference the cache pointer to access cache methods

	// Invoke cache interface since the last timestamp
	nodes := actual_cache.GetNodes(rc.timestamp, zero)
	latest_time := rc.timestamp

	for _, node := range nodes {
		timestamp, err := rc.updateNode(node)
		if err != nil {
			glog.Infof("Failed to Update Node Information: %s\n", err)
		}
		latest_time = maxTimestamp(latest_time, timestamp)
	}

	pods := actual_cache.GetPods(rc.timestamp, zero)
	for _, pod := range pods {
		timestamp, err := rc.updatePod(pod)
		if err != nil {
			glog.Infof("Failed to Update Pod Information: %s\n", err)
		}
		latest_time = maxTimestamp(latest_time, timestamp)
	}

	free_containers := actual_cache.GetFreeContainers(rc.timestamp, zero)
	for _, ce := range free_containers {
		timestamp, err := rc.updateFreeContainer(ce)
		if err != nil {
			glog.Infof("Failed to Update Free Container Information: %s\n", err)
		}
		latest_time = maxTimestamp(latest_time, timestamp)
	}

	// Update the Cluster timestamp to the latest time found in the new metrics
	rc.updateTime(latest_time)
	glog.V(2).Infoln("Schema Update operation completed")

	return nil
}

func (rc *realCluster) updateTime(new_time time.Time) {
	// Updates the Cluster timestamp to the specified time
	rc.lock.Lock()
	defer rc.lock.Unlock()
	rc.timestamp = new_time
}

func (rc *realCluster) updateNode(node_container *cache.ContainerElement) (time.Time, error) {
	// Inserts or updates Node information from "machine"-tagged containers
	name := node_container.Name
	var err error
	if name != "machine" {
		err := errors.New("Received node-level container with unexpected name")
		return time.Time{}, err
	}

	hostname := node_container.Hostname

	rc.lock.Lock()
	defer rc.lock.Unlock()
	node_ptr := rc.addNode(hostname)

	// Update NodeInfo's Metrics and Labels - return latest metric timestamp
	result, err := updateInfoType(&node_ptr.InfoType, node_container)

	return result, err
}

func (rc *realCluster) addNode(hostname string) *info.NodeInfo {
	//	Creates or finds a NodeInfo element for the provided hostname
	//	Assumes Cluster lock is already taken

	var node_ptr *info.NodeInfo

	if val, ok := rc.Nodes[hostname]; ok {
		// Node element already exists, return pointer
		node_ptr = val
	} else {
		// Node does not exist in map, create a new NodeInfo object
		node_ptr = &info.NodeInfo{
			InfoType:       newInfoType(nil, nil),
			Pods:           make(map[string]*info.PodInfo),
			FreeContainers: make(map[string]*info.ContainerInfo),
		}

		// Add Pointer to new_node under cluster.Nodes
		rc.Nodes[hostname] = node_ptr
	}
	return node_ptr
}

func (rc *realCluster) addNamespace(name string) *info.NamespaceInfo {
	//	Creates or finds a NamespaceInfo element for the provided namespace
	//	Assumes Cluster lock is already taken

	var namespace_ptr *info.NamespaceInfo

	if val, ok := rc.Namespaces[name]; ok {
		namespace_ptr = val
	} else {
		// Namespace does not exist in map, create a new NamespaceInfo struct
		namespace_ptr = &info.NamespaceInfo{
			InfoType: newInfoType(nil, nil),
			Pods:     make(map[string]*info.PodInfo),
		}
		rc.Namespaces[name] = namespace_ptr
	}

	return namespace_ptr
}

func (rc *realCluster) addPod(pod_name string, pod_uid string, namespace *info.NamespaceInfo, node *info.NodeInfo) *info.PodInfo {
	//	Creates or finds a PodInfo element given a Node and a Namespace
	//	Assumes Cluster lock is already taken

	var pod_ptr *info.PodInfo
	in_namespace := false
	in_node := false

	// Check if the pod is already referenced by the namespace or the node
	if _, ok := namespace.Pods[pod_name]; ok {
		in_namespace = true
	}

	if _, ok := node.Pods[pod_name]; ok {
		in_node = true
	}

	if in_namespace && in_node {
		// Pod already in Namespace and Node maps
		pod_ptr, _ = node.Pods[pod_name]
	} else if !in_namespace && !in_node {
		// Create new Pod and point from node and namespace
		pod_ptr = &info.PodInfo{
			InfoType:   newInfoType(nil, nil),
			UID:        pod_uid,
			Containers: make(map[string]*info.ContainerInfo),
		}
		namespace.Pods[pod_name] = pod_ptr
		node.Pods[pod_name] = pod_ptr
	}

	return pod_ptr
}

func (rc *realCluster) updatePod(pod *cache.PodElement) (time.Time, error) {
	// Appends Pod information in a new or existing PodInfo from a PodElement
	pod_name := pod.Name
	pod_hostname := pod.Hostname
	pod_uid := pod.UID
	pod_namespace := pod.Namespace

	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Get Namespace and Node pointers
	namespace := rc.addNamespace(pod_namespace)
	node := rc.addNode(pod_hostname)

	// Get Pod pointer
	pod_ptr := rc.addPod(pod_name, pod_uid, namespace, node)

	// Copy Labels pointer
	pod_ptr.Labels = pod.Labels

	// Update all container metrics
	latest_time := time.Time{}
	for _, ce := range pod.Containers {
		new_time, err := rc.updatePodContainer(pod_ptr, ce)
		if err != nil {
			return time.Time{}, err
		}
		latest_time = maxTimestamp(latest_time, new_time)
	}

	return latest_time, nil
}

func (rc *realCluster) updatePodContainer(pod_info *info.PodInfo, ce *cache.ContainerElement) (time.Time, error) {
	// Appends new data in a new or existing ContainerInfo under a specified PodInfo
	// Assumes Cluster lock is already taken

	cinfo := addContainerToMap(ce.Name, &pod_info.Containers)
	latest_time, err := updateInfoType(&cinfo.InfoType, ce)
	return latest_time, err
}

func (rc *realCluster) updateFreeContainer(ce *cache.ContainerElement) (time.Time, error) {
	// Inserts or updates Free Container data under the corresponding node
	rc.lock.Lock()
	defer rc.lock.Unlock()

	node := rc.addNode(ce.Hostname)
	cinfo := addContainerToMap(ce.Name, &node.FreeContainers)
	latest_time, err := updateInfoType(&cinfo.InfoType, ce)
	return latest_time, err
}
