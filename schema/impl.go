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
	"time"

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
)

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

// GetAllClusterData returns a pointer to the ClusterInfo, along with all of its metrics.
// GetAllClusterData also returns the latest cluster timestamp, for reuse in GetNew* methods.
func (rc *realCluster) GetAllClusterData() (*ClusterInfo, time.Time, error) {
	rc.lock.RLock()
	defer rc.lock.RUnlock()
	return &rc.ClusterInfo, rc.timestamp, nil
}

// GetAllNodeData finds a node, given a hostname (internal to the cluster).
// GetAllNodeData returns a corresponding NodeInfo object, along with all of its metrics.
// GetAllNodeData also returns the latest cluster timestamp, for reuse in GetNew* methods.
func (rc *realCluster) GetAllNodeData(hostname string) (*NodeInfo, time.Time, error) {
	// TODO(alex): should return a deep copy instead of a pointer
	var zeroTime time.Time

	rc.lock.RLock()
	defer rc.lock.RUnlock()

	res, ok := rc.Nodes[hostname]
	if !ok {
		return nil, zeroTime, fmt.Errorf("unable to find node with hostname: %s", hostname)
	}

	return res, rc.timestamp, nil
}

// GetAllPodData finds a pod, given a namespace string and a pod name string.
// GetAllPodData returns a pointer to a PodInfo object, along with all of its metrics.
// GetAllPodData also returns the latest cluster timestamp, for reuse in GetNew* methods.
func (rc *realCluster) GetAllPodData(namespace string, pod_name string) (*PodInfo, time.Time, error) {
	// TODO(alex): should return a deep copy instead of a pointer
	var zeroTime time.Time

	rc.lock.RLock()
	defer rc.lock.RUnlock()

	if len(rc.Namespaces) == 0 {
		return nil, zeroTime, fmt.Errorf("unable to find pod: no namespaces in cluster")
	}

	ns, ok := rc.Namespaces[namespace]
	if !ok {
		return nil, zeroTime, fmt.Errorf("unable to find namespace with name: %s", namespace)
	}

	pod, ok := ns.Pods[pod_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("unable to find pod with name: %s", pod_name)
	}

	return pod, rc.timestamp, nil
}

func (rc *realCluster) Update(c *cache.Cache) error {
	// TODO(afein): Unimplemented
	return nil
}

// updateTime updates the Cluster timestamp to the specified time.
func (rc *realCluster) updateTime(new_time time.Time) {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	rc.timestamp = new_time
}

// addNode creates or finds a NodeInfo element for the provided (internal) hostname.
// addNode returns a pointer to the NodeInfo element that was created or found.
// addNode assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) addNode(hostname string) *NodeInfo {
	var node_ptr *NodeInfo

	if val, ok := rc.Nodes[hostname]; ok {
		// Node element already exists, return pointer
		node_ptr = val
	} else {
		// Node does not exist in map, create a new NodeInfo object
		node_ptr = &NodeInfo{
			InfoType:       newInfoType(nil, nil),
			Pods:           make(map[string]*PodInfo),
			FreeContainers: make(map[string]*ContainerInfo),
		}

		// Add Pointer to new_node under cluster.Nodes
		rc.Nodes[hostname] = node_ptr
	}
	return node_ptr
}

// addNamespace creates or finds a NamespaceInfo element for the provided namespace.
// addNamespace returns a pointer to the NamespaceInfo element that was created or found.
// addNamespace assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) addNamespace(name string) *NamespaceInfo {
	var namespace_ptr *NamespaceInfo

	if val, ok := rc.Namespaces[name]; ok {
		// Namespace already exists, return pointer
		namespace_ptr = val
	} else {
		// Namespace does not exist in map, create a new NamespaceInfo struct
		namespace_ptr = &NamespaceInfo{
			InfoType: newInfoType(nil, nil),
			Pods:     make(map[string]*PodInfo),
		}
		rc.Namespaces[name] = namespace_ptr
	}

	return namespace_ptr
}

// addPod creates or finds a PodInfo element under the provided NodeInfo and NamespaceInfo.
// addPod returns a pointer to the PodInfo element that was created or found.
// addPod assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) addPod(pod_name string, pod_uid string, namespace *NamespaceInfo, node *NodeInfo) *PodInfo {
	var pod_ptr *PodInfo
	var in_ns bool
	var in_node bool

	// Check if the pod is already referenced by the namespace or the node
	if _, ok := namespace.Pods[pod_name]; ok {
		in_ns = true
	}

	if _, ok := node.Pods[pod_name]; ok {
		in_node = true
	}

	if in_ns && in_node {
		// Pod already in Namespace and Node maps, return pointer
		pod_ptr, _ = node.Pods[pod_name]
	} else {
		// Create new Pod and point from node and namespace
		pod_ptr = &PodInfo{
			InfoType:   newInfoType(nil, nil),
			UID:        pod_uid,
			Containers: make(map[string]*ContainerInfo),
		}
		namespace.Pods[pod_name] = pod_ptr
		node.Pods[pod_name] = pod_ptr
	}

	return pod_ptr
}
