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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/heapster/api/schema/info"
	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	cadvisor "github.com/google/cadvisor/info/v1"
	//fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
)

func TestNewCluster(t *testing.T) {
	// Tests the Cluster Interface Constructor
	cluster := NewCluster()
	assert.NotNil(t, cluster)
}

func TestNewRealCluster(t *testing.T) {
	// Tests the Cluster Implementation Constructor
	cluster := NewRealCluster()
	assert := assert.New(t)
	assert.NotNil(cluster)
	assert.NotNil(cluster.lock)
	assert.NotNil(cluster.timestamp)
	assert.NotNil(cluster.Metrics)
	assert.NotNil(cluster.Labels)
}

func TestUpdateFreeContainer(t *testing.T) {
	// Tests the flow of UpdateFreeContainer
	// Errors are asserted in the tests of the corresponding subroutines
	cluster := NewRealCluster()
	ce := ContainerElementFactory(nil)

	stamp, err := cluster.updateFreeContainer(ce)

	assert := assert.New(t)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.NotNil(cluster.Nodes[ce.Hostname])
	node := cluster.Nodes[ce.Hostname]
	assert.NotNil(node.FreeContainers[ce.Name])
	container := node.FreeContainers[ce.Name]
	assert.Empty(container.Labels)
	assert.NotEmpty(container.Metrics)
}

func TestUpdatePodContainer(t *testing.T) {
	// Tests the flow of UpdatePodContainer
	// Errors are asserted in the tests of the corresponding subroutines
	cluster := NewRealCluster()
	namespace := cluster.addNamespace("default")
	node := cluster.addNode("new_node_xyz")
	pod_ptr := cluster.addPod("new_pod", "1234-1245-235235", namespace, node)
	ce := ContainerElementFactory(nil)

	stamp, err := cluster.updatePodContainer(pod_ptr, ce)
	assert := assert.New(t)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.NotNil(pod_ptr.Containers[ce.Name])
}

func TestAddNamespace(t *testing.T) {
	// Tests all flows of addNamespace
	cluster := NewRealCluster()
	namespace_name := "notdefault"

	// First call : namespace does not exist
	namespace := cluster.addNamespace(namespace_name)

	assert := assert.New(t)
	assert.NotNil(namespace)
	assert.Equal(cluster.Namespaces[namespace_name], namespace)
	assert.NotNil(namespace.Metrics)
	assert.NotNil(namespace.Labels)
	assert.NotNil(namespace.Pods)

	// Second call : namespace already exists
	new_namespace := cluster.addNamespace(namespace_name)
	assert.Equal(new_namespace, namespace)
}

func TestAddNode(t *testing.T) {
	// Tests all flows of addNode
	cluster := NewRealCluster()
	hostname := "kubernetes-minion-xkhz"

	// First call : node does not exist
	node := cluster.addNode(hostname)

	assert := assert.New(t)
	assert.NotNil(node)
	assert.Equal(cluster.Nodes[hostname], node)
	assert.NotNil(node.Metrics)
	assert.NotNil(node.Labels)
	assert.NotNil(node.FreeContainers)
	assert.NotNil(node.Pods)

	// Second call : node already exists
	new_node := cluster.addNode(hostname)
	assert.Equal(new_node, node)
}

func TestAddPod(t *testing.T) {
	// Tests all flows of addPod
	cluster := NewRealCluster()
	pod_name := "podname-1243-xkhz"
	pod_uid := "123124-124124-124124124124"
	namespace := cluster.addNamespace("default")
	node := cluster.addNode("kubernetes-minion-xkhz")

	// First call : pod does not exist
	pod := cluster.addPod(pod_name, pod_uid, namespace, node)

	assert := assert.New(t)
	assert.NotNil(pod)
	assert.Equal(node.Pods[pod_name], pod)
	assert.Equal(namespace.Pods[pod_name], pod)
	assert.Equal(pod.UID, pod_uid)
	assert.NotNil(pod.Metrics)
	assert.NotNil(pod.Labels)
	assert.NotNil(pod.Containers)

	// Second call : pod already exists
	new_pod := cluster.addPod(pod_name, pod_uid, namespace, node)
	assert.NotNil(new_pod)
	assert.Equal(new_pod, pod)

	// Third call : References are mismatched
	other_node := cluster.addNode("other-kubernetes-minion")
	newest_pod := cluster.addPod(pod_name, pod_uid, namespace, other_node)
	assert.Nil(newest_pod)
}

func TestUpdateTime(t *testing.T) {
	// Tests the sanity of updateTime
	cluster := NewRealCluster()
	stamp := time.Now()
	cluster.updateTime(stamp)
	assert.Equal(t, cluster.timestamp, stamp)
}

func TestUpdatePod(t *testing.T) {
	// Tests the normal flow of updatePod
	cluster := NewRealCluster()
	pod_elem := PodElementFactory()
	stamp, err := cluster.updatePod(pod_elem)

	assert := assert.New(t)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.NotNil(cluster.Nodes[pod_elem.Hostname])
	pod_ptr := cluster.Nodes[pod_elem.Hostname].Pods[pod_elem.Name]
	assert.NotNil(cluster.Namespaces[pod_elem.Namespace])
	assert.Equal(pod_ptr, cluster.Namespaces[pod_elem.Namespace].Pods[pod_elem.Name])
	assert.Equal(pod_ptr.Labels, pod_elem.Labels)
}

func TestUpdateNodeInvalid(t *testing.T) {
	// Tests the error flow of updateNode, when an invalid machine container is passed
	cluster := NewRealCluster()
	ce := ContainerElementFactory(nil)
	stamp, err := cluster.updateNode(ce)

	assert := assert.New(t)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})
}

func TestUpdateNodeNormal(t *testing.T) {
	// Tests the normal flow of updateNode
	cluster := NewRealCluster()
	ce := ContainerElementFactory(nil)
	ce.Name = "machine"
	ce.Hostname = "dummy-minion-xkz"
	stamp, err := cluster.updateNode(ce)

	assert := assert.New(t)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
}

func TestUpdate(t *testing.T) {
	// Tests the normal flow of Update
	// Alternate flows all lead to glog.Fatal, and are thus not considered
	cluster := NewRealCluster()
	source_cache := CacheFactory()

	assert.NoError(t, cluster.Update(source_cache))
	verifyCacheFactoryCluster(&cluster.ClusterInfo, t)
}

func TestGetAllClusterData(t *testing.T) {
	// Tests the normal flow of GetAllClusterData
	cluster := NewCluster()
	source_cache := CacheFactory()
	assert := assert.New(t)

	assert.NoError(cluster.Update(source_cache))

	clinfo, stamp, err := cluster.GetAllClusterData()
	assert.NoError(err)
	assert.NotNil(clinfo)
	assert.NotEqual(stamp, time.Time{})
	verifyCacheFactoryCluster(clinfo, t)
}

func TestGetAllNodeData(t *testing.T) {
	// Tests all the flows of GetAllNodeData
	cluster := NewRealCluster()
	node_name := "hostname2"

	// First Call: Try to get a non-existing node
	res_node, stamp, err := cluster.GetAllNodeData(node_name)
	assert := assert.New(t)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})
	assert.Nil(res_node)

	// Second Call: Add the node to the Cluster and Get successfully
	cluster.Update(CacheFactory())
	res_node, stamp, err = cluster.GetAllNodeData(node_name)

	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.NotNil(res_node)
}

func TestGetAllPodData(t *testing.T) {
	// Tests the normal flow of GetAllPodData
	cluster := NewRealCluster()
	cluster.Update(CacheFactory())

	pod_name := "pod1"
	pod_hostname := "hostname2"
	pod_uid := "123"
	namespace := "test"
	res_pod, stamp, err := cluster.GetAllPodData(namespace, pod_name)

	assert := assert.New(t)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.Equal(res_pod.UID, pod_uid)
	assert.Equal(res_pod, cluster.Nodes[pod_hostname].Pods[pod_name])
}

func TestGetAllPodDataError(t *testing.T) {
	// Tests the error flows of GetAllPodData
	cluster := NewRealCluster()
	pod_name := "pod-xyz"
	namespace := "default"

	// First Call: No Namespaces in cluster
	res_pod, new_stamp, err := cluster.GetAllPodData(namespace, pod_name)

	assert := assert.New(t)
	assert.Error(err)
	assert.Equal(new_stamp, time.Time{})
	assert.Nil(res_pod)

	// Second Call: Namespace exists, pod does not exist under the specified namespace
	cluster.addNamespace(namespace)
	res_pod, new_stamp, err = cluster.GetAllPodData(namespace, pod_name)
	assert.Error(err)
	assert.Equal(new_stamp, time.Time{})
	assert.Nil(res_pod)

	// Third Call: Specific Namespace does not exist
	res_pod, new_stamp, err = cluster.GetAllPodData("other_namespace", pod_name)
	assert.Error(err)
	assert.Equal(new_stamp, time.Time{})
	assert.Nil(res_pod)
}

func PodElementFactory() *cache.PodElement {
	// Test Helper Function
	// Creates a PodElement with one ContainerElement and two ContainerMetricElements
	container_metadata := cache.Metadata{
		Name:      "test",
		Namespace: "default",
		UID:       "123123123",
		Hostname:  "testhost",
		Labels:    make(map[string]string),
	}
	metrics := []*cache.ContainerMetricElement{cmeFactory(), cmeFactory()}
	new_ce := &cache.ContainerElement{
		Metadata: container_metadata,
		Metrics:  metrics,
	}
	pod_metadata := cache.Metadata{
		Name:      "pod-xyz",
		Namespace: "default",
		UID:       "12312-124125-135135",
		Hostname:  "testhost",
		Labels:    make(map[string]string),
	}
	pod_ele := cache.PodElement{
		Metadata:   pod_metadata,
		Containers: []*cache.ContainerElement{new_ce},
	}
	return &pod_ele
}

func CacheFactory() *cache.Cache {
	// Test Helper Function
	// Creates a cache with a specific node/pod/container structure
	source_cache := cache.NewCache(time.Hour)

	// Generate 4 ContainerMetricElements
	cme_1 := cmeFactory()
	cme_2 := cmeFactory()
	cme_3 := cmeFactory()
	cme_4 := cmeFactory()

	// Generate a pod with two containers, and a pod without any containers
	container1 := source_api.Container{
		Name:     "container1",
		Hostname: "hostname2",
		Spec:     *cme_1.Spec,
		Stats:    []*cadvisor.ContainerStats{cme_1.Stats},
	}
	container2 := source_api.Container{
		Name:     "container2",
		Hostname: "hostname3",
		Spec:     *cme_2.Spec,
		Stats:    []*cadvisor.ContainerStats{cme_2.Stats},
	}

	containers := []source_api.Container{container1, container2}
	pods := []source_api.Pod{
		{
			PodMetadata: source_api.PodMetadata{
				Name:      "pod1",
				ID:        "123",
				Namespace: "test",
				Hostname:  "hostname2",
				Status:    "Running",
			},
			Containers: containers,
		},
		{
			PodMetadata: source_api.PodMetadata{
				Name:      "pod2",
				ID:        "1234",
				Namespace: "test",
				Hostname:  "hostname3",
				Status:    "Running",
			},
			Containers: []source_api.Container{},
		},
	}

	// Generate a machine container
	machine_container := source_api.Container{
		Name:     "/",
		Hostname: "hostname2",
		Spec:     *cme_3.Spec,
		Stats:    []*cadvisor.ContainerStats{cme_3.Stats},
	}
	// Generate a free container
	free_container := source_api.Container{
		Name:     "free_container1",
		Hostname: "hostname2",
		Spec:     *cme_4.Spec,
		Stats:    []*cadvisor.ContainerStats{cme_4.Stats},
	}

	other_containers := []source_api.Container{machine_container, free_container}

	//Enter everything in the cache
	source_cache.StorePods(pods)
	source_cache.StoreContainers(other_containers)

	return &source_cache
}

func verifyCacheFactoryCluster(clinfo *info.ClusterInfo, t *testing.T) {
	// Test Helper function
	// Performs assertions over a ClusterInfo populated with a CacheFactory instance
	assert := assert.New(t)
	assert.NotNil(clinfo.Nodes["hostname2"])
	node2 := clinfo.Nodes["hostname2"]
	assert.Len(node2.FreeContainers, 1)
	assert.NotNil(node2.FreeContainers["free_container1"])

	assert.NotNil(clinfo.Nodes["hostname3"])
	node3 := clinfo.Nodes["hostname3"]

	assert.NotNil(clinfo.Namespaces["test"])
	namespace := clinfo.Namespaces["test"]

	assert.NotNil(namespace.Pods)
	pod1_ptr := namespace.Pods["pod1"]
	assert.Equal(pod1_ptr, node2.Pods["pod1"])
	assert.Len(pod1_ptr.Containers, 2)
	pod2_ptr := namespace.Pods["pod2"]
	assert.Equal(pod2_ptr, node3.Pods["pod2"])
	assert.Len(pod2_ptr.Containers, 0)
}
