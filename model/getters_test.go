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

package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetClusterMetric tests all flows of GetClusterMetric.
func TestGetClusterMetric(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		zeroTime     = time.Time{}
		dummyRequest = MetricRequest{
			MetricName: cpuUsage,
			Start:      zeroTime,
			End:        zeroTime,
		}
	)
	// Invocation with no cluster metrics
	res, stamp, err := cluster.GetClusterMetric(ClusterMetricRequest{
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant metric
	dummyRequest.MetricName = "doesnotexist"
	res, stamp, err = cluster.GetClusterMetric(ClusterMetricRequest{
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	dummyRequest.MetricName = memUsage
	res, stamp, err = cluster.GetClusterMetric(ClusterMetricRequest{
		MetricRequest: dummyRequest,
	})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetNodeMetric tests all flows of GetNodeMetric.
func TestGetNodeMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		hostname     = "hostname3"
		dummyRequest = MetricRequest{
			MetricName: cpuUsage,
			Start:      zeroTime,
			End:        zeroTime,
		}
	)
	// Invocation with no nodes in cluster
	res, stamp, err := cluster.GetNodeMetric(NodeMetricRequest{
		NodeName:      hostname,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant node
	res, stamp, err = cluster.GetNodeMetric(NodeMetricRequest{
		NodeName:      "doesnotexist",
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new node - no metrics
	cluster.addNode("newnode")
	res, stamp, err = cluster.GetNodeMetric(NodeMetricRequest{
		NodeName:      "newnode",
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	dummyRequest.MetricName = "doesnotexist"
	res, stamp, err = cluster.GetNodeMetric(NodeMetricRequest{
		NodeName:      hostname,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	dummyRequest.MetricName = memUsage
	res, stamp, err = cluster.GetNodeMetric(NodeMetricRequest{
		NodeName:      hostname,
		MetricRequest: dummyRequest,
	})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetNamespaceMetric tests all flows of GetNamespaceMetric.
func TestGetNamespaceMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		namespace    = "test"
		dummyRequest = MetricRequest{
			MetricName: cpuUsage,
			Start:      zeroTime,
			End:        zeroTime,
		}
	)
	// Invocation with no namespaces in cluster
	res, stamp, err := cluster.GetNamespaceMetric(NamespaceMetricRequest{
		NamespaceName: namespace,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant namespace
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceMetricRequest{
		NamespaceName: "doesnotexist",
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new namespace - no metrics
	cluster.addNamespace("newns")
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceMetricRequest{
		NamespaceName: "newns",
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	dummyRequest.MetricName = "doesnotexist"
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceMetricRequest{
		NamespaceName: namespace,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	dummyRequest.MetricName = memUsage
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceMetricRequest{
		NamespaceName: namespace,
		MetricRequest: dummyRequest,
	})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetPodMetric tests all flows of GetPodMetric.
func TestGetPodMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		namespace    = "test"
		pod          = "pod1"
		node         = "hostname2"
		dummyRequest = MetricRequest{
			MetricName: cpuUsage,
			Start:      zeroTime,
			End:        zeroTime,
		}
	)
	// Invocation with no namespaces in cluster
	res, stamp, err := cluster.GetPodMetric(PodMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant namespace
	res, stamp, err = cluster.GetPodMetric(PodMetricRequest{
		NamespaceName: "doesnotexist",
		PodName:       pod,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant pod
	res, stamp, err = cluster.GetPodMetric(PodMetricRequest{
		NamespaceName: namespace,
		PodName:       "otherpod",
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new pod - no metrics
	cluster.addPod(pod, "1234", cluster.Namespaces[namespace], cluster.Nodes[node])
	res, stamp, err = cluster.GetPodMetric(PodMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	dummyRequest.MetricName = "doesnotexist"
	res, stamp, err = cluster.GetPodMetric(PodMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	dummyRequest.MetricName = memUsage
	res, stamp, err = cluster.GetPodMetric(PodMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		MetricRequest: dummyRequest,
	})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetPodContainerMetric tests all flows of GetPodContainerMetric.
func TestGetPodContainerMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		namespace    = "test"
		pod          = "pod1"
		container    = "container1"
		dummyRequest = MetricRequest{
			MetricName: cpuUsage,
			Start:      zeroTime,
			End:        zeroTime,
		}
	)
	// Invocation with no namespaces in cluster
	res, stamp, err := cluster.GetPodContainerMetric(PodContainerMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant namespace
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerMetricRequest{
		NamespaceName: "doesnotexist",
		PodName:       pod,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new namespace - no metrics
	cluster.addNamespace("newns")
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerMetricRequest{
		NamespaceName: "newns",
		PodName:       pod,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant pod
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerMetricRequest{
		NamespaceName: namespace,
		PodName:       "otherpod",
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant container
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		ContainerName: "doesnotexist",
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	dummyRequest.MetricName = "doesnotexist"
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	dummyRequest.MetricName = memUsage
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerMetricRequest{
		NamespaceName: namespace,
		PodName:       pod,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetFreeContainerMetric tests all flows of GetFreeContainerMetric.
func TestGetFreeContainerMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		node         = "hostname2"
		container    = "free_container1"
		dummyRequest = MetricRequest{
			MetricName: cpuUsage,
			Start:      zeroTime,
			End:        zeroTime,
		}
	)
	// Invocation with no nodes in cluster
	res, stamp, err := cluster.GetFreeContainerMetric(FreeContainerMetricRequest{
		NodeName:      node,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant node
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerMetricRequest{
		NodeName:      "doesnotexist",
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new node - no metrics
	cluster.addNode("newnode")
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerMetricRequest{
		NodeName:      "newnode",
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant container
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerMetricRequest{
		NodeName:      node,
		ContainerName: "not_actual_ctr",
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	dummyRequest.MetricName = "doesnotexist"
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerMetricRequest{
		NodeName:      node,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	dummyRequest.MetricName = memUsage
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerMetricRequest{
		NodeName:      node,
		ContainerName: container,
		MetricRequest: dummyRequest,
	})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

//TestGetNodes tests the flow of GetNodes.
func TestGetNodes(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	// Invocation with empty cluster
	res := cluster.GetNodes()
	assert.Len(res, 0)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Normal Invocation
	res = cluster.GetNodes()
	assert.Len(res, 2)
}

//TestGetNamespaces tests the flow of GetNamespaces.
func TestGetNamespaces(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	// Invocation with empty cluster
	res := cluster.GetNamespaces()
	assert.Len(res, 0)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Normal Invocation
	res = cluster.GetNamespaces()
	assert.Len(res, 1)
}

//TestGetPods tests the flow of GetPods.
func TestGetPods(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	// Invocation with empty cluster
	res := cluster.GetPods("test")
	assert.Len(res, 0)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Normal Invocation
	res = cluster.GetPods("test")
	assert.Len(res, 2)

	// Invocation with non-existant namespace
	res = cluster.GetPods("fakenamespace")
	assert.Len(res, 0)
}

//TestGetPodContainers tests the flow of GetPodContainers.
func TestGetPodContainers(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	// Invocation with empty cluster
	res := cluster.GetPodContainers("test", "pod1")
	assert.Len(res, 0)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Normal Invocation
	res = cluster.GetPodContainers("test", "pod1")
	assert.Len(res, 2)

	// Invocation with non-existant namespace
	res = cluster.GetPodContainers("fail", "pod1")
	assert.Len(res, 0)

	// Invocation with non-existant pod
	res = cluster.GetPodContainers("test", "pod5")
	assert.Len(res, 0)
}

//TestGetFreeContainers tests the flow of GetFreeContainers.
func TestGetFreeContainers(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	// Invocation with empty cluster
	res := cluster.GetFreeContainers("hostname2")
	assert.Len(res, 0)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Normal Invocation
	res = cluster.GetFreeContainers("hostname2")
	assert.Len(res, 1)

	// Invocation with non-existant node
	res = cluster.GetFreeContainers("hostname9")
	assert.Len(res, 0)
}

// TestGetAvailableMetrics tests the flow of GetAvailableMetrics.
func TestGetAvailableMetrics(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	// Invocation with no cluster metrics
	res := cluster.GetAvailableMetrics()
	assert.Len(res, 0)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Normal Invocation
	res = cluster.GetAvailableMetrics()
	assert.Len(res, 7)
}

// TestGetClusterStats tests all flows of GetClusterStats.
func TestGetClusterStats(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)

	// Invocation with no cluster stats
	res, uptime, err := cluster.GetClusterStats()
	assert.Len(res, 0)
	assert.Equal(uptime, time.Duration(0))
	assert.NoError(err)

	// Invocation with cluster stats
	cluster.Update(source_cache)
	res, uptime, err = cluster.GetClusterStats()
	assert.Len(res, 7)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Max)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Percentile)
	assert.Equal(res[memUsage].Minute.Average, uint64(10000))
	assert.Equal(res[memUsage].Hour.Max, uint64(10000))
	assert.Equal(res[memUsage].Hour.Average, uint64((5000+10000)/2))
	assert.Equal(res[memUsage].Hour.Percentile, uint64(10000))
	assert.NotEqual(uptime, time.Duration(0))
	assert.NoError(err)
}

// TestGetNodeStats tests all flows of GetNodeStats.
func TestGetNodeStats(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	cluster.Update(source_cache)

	// Invocation with nonexistant node
	res, uptime, err := cluster.GetNodeStats(NodeRequest{
		NodeName: "nonexistant",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Invocation with normal node
	res, uptime, err = cluster.GetNodeStats(NodeRequest{
		NodeName: "hostname2",
	})
	assert.Len(res, 6) // No cpu usage for hostname2
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Max)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Percentile)
	assert.NotEqual(res[memUsage].Minute.Average, uint64(0))
	assert.NotEqual(res[memUsage].Hour.Average, uint64(0))
	assert.NoError(err)
}

// TestGetNamespaceStats tests all flows of GetNamespaceStats.
func TestGetNamespaceStats(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	cluster.Update(source_cache)

	// Invocation with nonexistant namespace
	res, uptime, err := cluster.GetNamespaceStats(NamespaceRequest{
		NamespaceName: "nonexistant",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Invocation with normal namespace
	res, uptime, err = cluster.GetNamespaceStats(NamespaceRequest{
		NamespaceName: "test",
	})
	assert.Len(res, 6)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Max)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Percentile)
	assert.NotEqual(res[memUsage].Minute.Average, uint64(0))
	assert.NotEqual(res[memUsage].Hour.Average, uint64(0))
	assert.NoError(err)
}

// TestGetPodStats tests all flows of GetPodStats.
func TestGetPodStats(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	cluster.Update(source_cache)

	// Invocation with nonexistant namespace
	res, uptime, err := cluster.GetPodStats(PodRequest{
		NamespaceName: "nonexistant",
		PodName:       "pod1",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Invocation with normal namespace and nonexistant pod
	res, uptime, err = cluster.GetPodStats(PodRequest{
		NamespaceName: "test",
		PodName:       "nonexistant",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Normal Invocation
	res, uptime, err = cluster.GetPodStats(PodRequest{
		NamespaceName: "test",
		PodName:       "pod1",
	})
	assert.Len(res, 6)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Max)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Percentile)
	assert.NotEqual(res[memUsage].Minute.Average, uint64(0))
	assert.NotEqual(res[memUsage].Hour.Average, uint64(0))
	assert.NoError(err)
}

// TestGetPodContainerStats tests all flows of GetPodContainerStats.
func TestGetPodContainerStats(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	cluster.Update(source_cache)

	// Invocation with nonexistant namespace
	res, uptime, err := cluster.GetPodContainerStats(PodContainerRequest{
		NamespaceName: "nonexistant",
		PodName:       "pod1",
		ContainerName: "container1",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Invocation with normal namespace and nonexistant pod
	res, uptime, err = cluster.GetPodContainerStats(PodContainerRequest{
		NamespaceName: "test",
		PodName:       "nonexistant",
		ContainerName: "container1",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Invocation with normal namespace and pod, but nonexistant container
	res, uptime, err = cluster.GetPodContainerStats(PodContainerRequest{
		NamespaceName: "test",
		PodName:       "pod1",
		ContainerName: "nonexistant",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Normal Invocation
	res, uptime, err = cluster.GetPodContainerStats(PodContainerRequest{
		NamespaceName: "test",
		PodName:       "pod1",
		ContainerName: "container1",
	})
	assert.Len(res, 6)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Max)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Percentile)
	assert.NotEqual(res[memUsage].Minute.Average, uint64(0))
	assert.NotEqual(res[memUsage].Hour.Average, uint64(0))
	assert.NoError(err)
}

// TestGetFreeContainerStats tests all flows of GetFreeContainerStats.
func TestGetFreeContainerStats(t *testing.T) {
	var (
		cluster      = newRealCluster(newDayStore, newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
	)
	cluster.Update(source_cache)

	// Invocation with nonexistant node
	res, uptime, err := cluster.GetFreeContainerStats(FreeContainerRequest{
		NodeName:      "nonexistant",
		ContainerName: "free_container1",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Invocation with normal node and nonexistant container
	res, uptime, err = cluster.GetFreeContainerStats(FreeContainerRequest{
		NodeName:      "hostname2",
		ContainerName: "nonexistant",
	})
	assert.Nil(res)
	assert.Equal(uptime, time.Duration(0))
	assert.Error(err)

	// Normal Invocation
	res, uptime, err = cluster.GetFreeContainerStats(FreeContainerRequest{
		NodeName:      "hostname2",
		ContainerName: "free_container1",
	})
	assert.Len(res, 6)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Max)
	assert.Equal(res[memUsage].Minute.Average, res[memUsage].Minute.Percentile)
	assert.NotEqual(res[memUsage].Minute.Average, uint64(0))
	assert.NotEqual(res[memUsage].Hour.Average, uint64(0))
	assert.NoError(err)
}
