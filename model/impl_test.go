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
	"math/rand"
	"strings"
	"testing"
	"time"

	cadvisor "github.com/google/cadvisor/info/v1"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/store"
)

// newTimeStore creates a new GCStore and returns it as a TimeStore.
// Meant to be passed to newRealCluster calls in all unit tests.
func newTimeStore() store.TimeStore {
	return store.NewGCStore(store.NewCMAStore(), 24*time.Hour)
}

// TestNewCluster tests the sanity of NewCluster
func TestNewCluster(t *testing.T) {
	cluster := NewCluster(newTimeStore, time.Minute)
	assert.NotNil(t, cluster)
}

// TestAddNamespace tests all flows of addNamespace.
func TestAddNamespace(t *testing.T) {
	var (
		cluster        = newRealCluster(newTimeStore, time.Minute)
		namespace_name = "default"
		assert         = assert.New(t)
	)

	// First call : namespace does not exist
	namespace := cluster.addNamespace(namespace_name)

	assert.NotNil(namespace)
	assert.Equal(cluster.Namespaces[namespace_name], namespace)
	assert.NotNil(namespace.Metrics)
	assert.NotNil(namespace.Labels)
	assert.NotNil(namespace.Pods)

	// Second call : namespace already exists
	new_namespace := cluster.addNamespace(namespace_name)
	assert.Equal(new_namespace, namespace)
}

// TestAddNode tests all flows of addNode.
func TestAddNode(t *testing.T) {
	var (
		cluster  = newRealCluster(newTimeStore, time.Minute)
		hostname = "kubernetes-minion-xkhz"
		assert   = assert.New(t)
	)

	// First call : node does not exist
	node := cluster.addNode(hostname)

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

// TestAddPod tests all flows of addPod.
func TestAddPod(t *testing.T) {
	var (
		cluster   = newRealCluster(newTimeStore, time.Minute)
		pod_name  = "podname-xkhz"
		pod_uid   = "123124-124124-124124124124"
		namespace = cluster.addNamespace("default")
		node      = cluster.addNode("kubernetes-minion-xkhz")
		assert    = assert.New(t)
	)

	// First call : pod does not exist
	pod := cluster.addPod(pod_name, pod_uid, namespace, node)

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

	// Third call : Nil namespace
	new_pod = cluster.addPod(pod_name, pod_uid, nil, node)
	assert.Nil(new_pod)

	// Fourth call : Nil node
	new_pod = cluster.addPod(pod_name, pod_uid, namespace, nil)
	assert.Nil(new_pod)
}

// TestUpdateTime tests the sanity of updateTime.
func TestUpdateTime(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		stamp   = time.Now()
	)

	// First call: update with non-zero time
	cluster.updateTime(stamp)
	assert.Equal(t, cluster.timestamp, stamp)

	// Second call: update with zero time
	cluster.updateTime(time.Time{})
	assert.Equal(t, cluster.timestamp, stamp)
}

// Tests the flow of AddMetricToMap where the metric name is present in the map
func TestAddMetricToMapExistingKey(t *testing.T) {
	var (
		cluster         = newRealCluster(newTimeStore, time.Minute)
		metrics         = make(map[string]*store.TimeStore)
		new_metric_name = "name_already_in_map"
		value           = uint64(1234567890)
		zeroTime        = time.Time{}
		stamp           = time.Now().Round(time.Minute)
		assert          = assert.New(t)
	)

	// Fist Call: addMetricToMap for a new metric
	assert.NoError(cluster.addMetricToMap(new_metric_name, stamp, value, metrics))

	ts := *metrics[new_metric_name]
	results := ts.Get(zeroTime, zeroTime)
	assert.Len(results, 1)
	assert.Equal(results[0].Timestamp, stamp)
	assert.Equal(results[0].Value, value)

	// Second Call: addMetricToMap for an existing key, same time
	new_value := uint64(102)
	assert.NoError(cluster.addMetricToMap(new_metric_name, stamp, new_value, metrics))

	ts = *metrics[new_metric_name]
	results = ts.Get(zeroTime, zeroTime)
	assert.Len(results, 1)
	assert.Equal(results[0].Timestamp, stamp)
	assert.Equal(results[0].Value, uint64(617283996))

	// Second Call: addMetricToMap for an existing key, new time
	later_stamp := stamp.Add(2 * time.Hour)
	assert.NoError(cluster.addMetricToMap(new_metric_name, later_stamp, new_value, metrics))

	ts = *metrics[new_metric_name]
	results = ts.Get(zeroTime, zeroTime)
	assert.Len(results, 2)
	assert.Equal(results[0].Timestamp, later_stamp)
	assert.Equal(results[0].Value, uint64(102))
	assert.Equal(results[1].Timestamp, stamp)
	assert.Equal(results[1].Value, uint64(617283996))

	// Third Call: addMetricToMap for an existing key, overwrites previous data
	later_stamp = stamp.Add(48 * time.Hour)
	assert.NoError(cluster.addMetricToMap(new_metric_name, later_stamp, new_value, metrics))

	ts = *metrics[new_metric_name]
	results = ts.Get(zeroTime, zeroTime)
	assert.Len(results, 1)
	assert.Equal(results[0].Timestamp, later_stamp)
	assert.Equal(results[0].Value, uint64(102))

	// Fourth Call: addMetricToMap for an existing key, cause TimeStore failure
	assert.Error(cluster.addMetricToMap(new_metric_name, zeroTime, new_value, metrics))
}

// Tests the flow of AddMetricToMap where the metric name is not present in the map
func TestAddMetricToMapNewKey(t *testing.T) {
	var (
		cluster         = newRealCluster(newTimeStore, time.Minute)
		metrics         = make(map[string]*store.TimeStore)
		new_metric_name = "name_not_in_map"
		stamp           = time.Now()
		zeroTime        = time.Time{}
		value           = uint64(1234567890)
		assert          = assert.New(t)
	)

	// First Call: Add a new metric to the map
	assert.NoError(cluster.addMetricToMap(new_metric_name, stamp, value, metrics))
	new_ts := *metrics[new_metric_name]
	results := new_ts.Get(zeroTime, zeroTime)
	assert.Len(results, 1)
	assert.Equal(results[0].Timestamp, stamp)
	assert.Equal(results[0].Value, value)

	// Second Call: addMetricToMap for a new key, cause TimeStore failure
	assert.Error(cluster.addMetricToMap("other_metric", zeroTime, value, metrics))
}

// TestParseMetricError tests the error flows of ParseMetric
func TestParseMetricError(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		context = make(map[string]*store.TimePoint)
		dict    = make(map[string]*store.TimeStore)
		cme     = cmeFactory()
		assert  = assert.New(t)
	)

	// Invoke parseMetric with a nil cme argument
	stamp, err := cluster.parseMetric(nil, dict, context)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})

	// Invoke parseMetric with a nil dict argument
	stamp, err = cluster.parseMetric(cme, nil, context)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})

	// Invoke parseMetric with a nil context argument
	stamp, err = cluster.parseMetric(cme, dict, nil)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})
}

// TestParseMetricNormal tests the normal flow of ParseMetric
func TestParseMetricNormal(t *testing.T) {
	var (
		cluster    = newRealCluster(newTimeStore, time.Minute)
		zeroTime   = time.Time{}
		metrics    = make(map[string]*store.TimeStore)
		context    = make(map[string]*store.TimePoint)
		normal_cme = cmeFactory()
		other_cme  = cmeFactory()
		assert     = assert.New(t)
	)
	normal_stamp := normal_cme.Stats.Timestamp.Round(time.Minute)
	normal_cme.Stats.Cpu.Usage.Total = uint64(1000)

	other_cme.Stats.Timestamp = normal_stamp.Add(3 * time.Minute)
	other_cme.Stats.Cpu.Usage.Total = uint64(360000001000) // 2 stable over 3 minutes in NS
	other_stamp := other_cme.Stats.Timestamp.Round(time.Minute)

	// Normal Invocation with a regular CME, passed twice
	stamp, err := cluster.parseMetric(normal_cme, metrics, context)
	assert.NoError(err)
	assert.Equal(stamp, normal_stamp)
	stamp, err = cluster.parseMetric(other_cme, metrics, context)
	assert.NoError(err)
	assert.Equal(stamp, other_stamp)
	for key, ts := range metrics {
		actual_ts := *ts
		pointSlice := actual_ts.Get(zeroTime, zeroTime)
		metric := pointSlice[0]
		switch key {
		case cpuLimit:
			assert.Len(pointSlice, 2)
			assert.Equal(metric.Timestamp, other_stamp)
			assert.Equal(metric.Value, other_cme.Spec.Cpu.Limit)
			metric = pointSlice[1]
			assert.Equal(metric.Timestamp, normal_stamp)
			assert.Equal(metric.Value, normal_cme.Spec.Cpu.Limit)
		case memLimit:
			assert.Len(pointSlice, 2)
			assert.Equal(metric.Timestamp, other_stamp)
			assert.Equal(metric.Value, other_cme.Spec.Memory.Limit)
			metric = pointSlice[1]
			assert.Equal(metric.Timestamp, normal_stamp)
			assert.Equal(metric.Value, normal_cme.Spec.Memory.Limit)
		case cpuUsage:
			assert.Len(pointSlice, 1)
			assert.Equal(metric.Timestamp, other_stamp)
			assert.Equal(metric.Value.(uint64), 2*other_cme.Spec.Cpu.Limit)
		case memUsage:
			assert.Len(pointSlice, 2)
			assert.Equal(metric.Timestamp, other_stamp)
			assert.Equal(metric.Value, other_cme.Stats.Memory.Usage)
			metric = pointSlice[1]
			assert.Equal(metric.Timestamp, normal_stamp)
			assert.Equal(metric.Value, normal_cme.Stats.Memory.Usage)
		case memWorking:
			assert.Len(pointSlice, 2)
			assert.Equal(metric.Timestamp, other_stamp)
			assert.Equal(metric.Value, other_cme.Stats.Memory.WorkingSet)
			metric = pointSlice[1]
			assert.Equal(metric.Timestamp, normal_stamp)
			assert.Equal(metric.Value, normal_cme.Stats.Memory.WorkingSet)
		default:
			// Filesystem or error
			assert.Len(pointSlice, 2)
			if strings.Contains(key, "limit") {
				assert.Equal(metric.Value, other_cme.Stats.Filesystem[0].Limit)
			} else if strings.Contains(key, "usage") {
				assert.Equal(metric.Value, other_cme.Stats.Filesystem[0].Usage)
			} else {
				assert.True(false, "Unknown key in resulting metrics slice")
			}
		}
	}
}

// TestUpdateInfoTypeError Tests the error flows of updateInfoType.
func TestUpdateInfoTypeError(t *testing.T) {
	var (
		cluster      = newRealCluster(newTimeStore, time.Minute)
		new_infotype = newInfoType(nil, nil, nil)
		full_ce      = containerElementFactory(nil)
		assert       = assert.New(t)
	)

	// Invocation with a nil InfoType argument
	stamp, err := cluster.updateInfoType(nil, full_ce)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})

	// Invocation with a nil ContainerElement argument
	stamp, err = cluster.updateInfoType(&new_infotype, nil)
	assert.Error(err)
	assert.Empty(new_infotype.Metrics)
	assert.Equal(stamp, time.Time{})
}

// TestUpdateInfoTypeNormal tests the normal flows of UpdateInfoType.
func TestUpdateInfoTypeNormal(t *testing.T) {
	var (
		cluster      = newRealCluster(newTimeStore, time.Minute)
		new_cme      = cmeFactory()
		empty_ce     = containerElementFactory([]*cache.ContainerMetricElement{})
		nil_ce       = containerElementFactory([]*cache.ContainerMetricElement{new_cme, nil})
		new_infotype = newInfoType(nil, nil, nil)
		zeroTime     = time.Time{}
		assert       = assert.New(t)
	)

	// Invocation with a ContainerElement argument with no CMEs
	stamp, err := cluster.updateInfoType(&new_infotype, empty_ce)
	assert.NoError(err)
	assert.Empty(new_infotype.Metrics)
	assert.Equal(stamp, time.Time{})

	// Invocation with a ContainerElement argument with a nil CME
	stamp, err = cluster.updateInfoType(&new_infotype, nil_ce)
	assert.NoError(err)
	assert.NotEmpty(new_infotype.Metrics)
	assert.NotEqual(stamp, time.Time{})
	assert.Len(new_infotype.Metrics, 6) // 6 stats in total - no CPU Usage yet
	for _, metricStore := range new_infotype.Metrics {
		metricSlice := (*metricStore).Get(zeroTime, zeroTime)
		assert.Len(metricSlice, 1) // 1 Metric per stat
	}

	// Invocation with an empty InfoType argument
	// The new ContainerElement adds one TimePoint to each of 7 Metrics
	newer_cme := cmeFactory()
	newer_cme.Stats.Timestamp = new_cme.Stats.Timestamp.Add(time.Hour)
	newer_cme.Stats.Cpu.Usage.Total = new_cme.Stats.Cpu.Usage.Total + uint64(3600000000)
	new_ce := containerElementFactory([]*cache.ContainerMetricElement{newer_cme})
	stamp, err = cluster.updateInfoType(&new_infotype, new_ce)
	assert.NoError(err)
	assert.Empty(new_infotype.Labels)
	assert.NotEqual(stamp, time.Time{})
	assert.Len(new_infotype.Metrics, 7) // 7 stats in total
	for key, metricStore := range new_infotype.Metrics {
		metricSlice := (*metricStore).Get(zeroTime, zeroTime)
		if key == cpuUsage {
			assert.Len(metricSlice, 1) // cpuUsage has n-1 values.
		} else {
			assert.Len(metricSlice, 2) // 2 Metrics per stat
		}
	}

	// Invocation with an existing infotype as argument
	// The new ContainerElement adds two TimePoints to each Metric
	newer_cme2 := cmeFactory()
	newer_cme2.Stats.Timestamp = newer_cme.Stats.Timestamp.Add(time.Hour)
	newer_cme2.Stats.Cpu.Usage.Total = newer_cme.Stats.Cpu.Usage.Total + uint64(3600000000)
	newer_cme3 := cmeFactory()
	newer_cme3.Stats.Timestamp = newer_cme2.Stats.Timestamp.Add(time.Hour)
	newer_cme3.Stats.Cpu.Usage.Total = newer_cme2.Stats.Cpu.Usage.Total + uint64(360000000)
	new_ce = containerElementFactory([]*cache.ContainerMetricElement{newer_cme3, newer_cme2})
	stamp, err = cluster.updateInfoType(&new_infotype, new_ce)
	assert.NoError(err)
	assert.Empty(new_infotype.Labels)
	assert.NotEqual(stamp, time.Time{})
	assert.Len(new_infotype.Metrics, 7) // 7 stats total
	for key, metricStore := range new_infotype.Metrics {
		metricSlice := (*metricStore).Get(zeroTime, zeroTime)
		if key == cpuUsage {
			assert.Len(metricSlice, 3) // cpuUsage consists of n-1 values.
		} else {
			assert.Len(metricSlice, 4) // 4 Metrics per stat
		}
	}
}

// TestUpdateFreeContainer tests the flow of updateFreeContainer
func TestUpdateFreeContainer(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		ce      = containerElementFactory(nil)
		assert  = assert.New(t)
	)

	// Invocation with regular parameters
	stamp, err := cluster.updateFreeContainer(ce)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.NotNil(cluster.Nodes[ce.Hostname])
	node := cluster.Nodes[ce.Hostname]
	assert.NotNil(node.FreeContainers[ce.Name])
	container := node.FreeContainers[ce.Name]
	assert.Empty(container.Labels)
	assert.NotEmpty(container.Metrics)
}

// TestUpdatePodContainer tests the flow of updatePodContainer
func TestUpdatePodContainer(t *testing.T) {
	var (
		cluster   = newRealCluster(newTimeStore, time.Minute)
		namespace = cluster.addNamespace("default")
		node      = cluster.addNode("new_node_xyz")
		pod_ptr   = cluster.addPod("new_pod", "1234-1245-235235", namespace, node)
		ce        = containerElementFactory(nil)
		assert    = assert.New(t)
	)

	// Invocation with regular parameters
	stamp, err := cluster.updatePodContainer(pod_ptr, ce)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.NotNil(pod_ptr.Containers[ce.Name])
}

// TestUpdatePodNormal tests the normal flow of updatePod.
func TestUpdatePodNormal(t *testing.T) {
	var (
		cluster  = newRealCluster(newTimeStore, time.Minute)
		pod_elem = podElementFactory()
		assert   = assert.New(t)
	)
	// Invocation with a regular parameter
	stamp, err := cluster.updatePod(pod_elem)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
	assert.NotNil(cluster.Nodes[pod_elem.Hostname])
	pod_ptr := cluster.Nodes[pod_elem.Hostname].Pods[pod_elem.Name]
	assert.NotNil(cluster.Namespaces[pod_elem.Namespace])
	assert.Equal(pod_ptr, cluster.Namespaces[pod_elem.Namespace].Pods[pod_elem.Name])
	assert.Equal(pod_ptr.Labels, pod_elem.Labels)
}

// TestUpdatePodError tests the error flow of updatePod.
func TestUpdatePodError(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		assert  = assert.New(t)
	)
	// Invocation with a nil parameter
	stamp, err := cluster.updatePod(nil)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})
}

// TestUpdateNodeInvalid tests the error flow of updateNode.
func TestUpdateNodeInvalid(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		ce      = containerElementFactory(nil)
		assert  = assert.New(t)
	)

	// Invocation with a ContainerElement that is not "machine"-tagged
	stamp, err := cluster.updateNode(ce)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})
}

// TestUpdateNodeNormal tests the normal flow of updateNode.
func TestUpdateNodeNormal(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		ce      = containerElementFactory(nil)
		assert  = assert.New(t)
	)
	ce.Name = "machine"
	ce.Hostname = "dummy-minion-xkz"

	// Invocation with regular parameters
	stamp, err := cluster.updateNode(ce)
	assert.NoError(err)
	assert.NotEqual(stamp, time.Time{})
}

// TestUpdate tests the normal flows of Update.
// TestUpdate performs consecutive calls to Update with both empty and non-empty caches
func TestUpdate(t *testing.T) {
	var (
		cluster      = newRealCluster(newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		empty_cache  = cache.NewCache(24 * time.Hour)
		zeroTime     = time.Time{}
	)

	// Invocation with empty cache
	assert.NoError(cluster.Update(empty_cache))
	assert.Empty(cluster.Nodes)
	assert.Empty(cluster.Namespaces)
	assert.Empty(cluster.Metrics)

	// Invocation with regular parameters
	assert.NoError(cluster.Update(source_cache))
	verifyCacheFactoryCluster(&cluster.ClusterInfo, t)

	// Assert Node Metric aggregation
	assert.NotEmpty(cluster.Nodes)
	assert.NotEmpty(cluster.Metrics)
	assert.NotNil(cluster.Metrics[memWorking])
	mem_work_ts := *(cluster.Metrics[memWorking])
	actual := mem_work_ts.Get(zeroTime, zeroTime)
	assert.Len(actual, 2)
	// Datapoint present in both nodes, added up to 1024
	assert.Equal(actual[1].Value.(uint64), uint64(1204))
	// Datapoint present in only one node
	assert.Equal(actual[0].Value.(uint64), uint64(602))

	assert.NotNil(cluster.Metrics[memUsage])
	mem_usage_ts := *(cluster.Metrics[memUsage])
	actual = mem_usage_ts.Get(zeroTime, zeroTime)
	assert.Len(actual, 2)
	// Datapoint present in both nodes, added up to 10000
	assert.Equal(actual[1].Value.(uint64), uint64(10000))
	// Datapoint present in only one node
	assert.Equal(actual[0].Value.(uint64), uint64(5000))

	// Assert Kubernetes Metric aggregation up to namespaces
	ns := cluster.Namespaces["test"]
	mem_work_ts = *(ns.Metrics[memWorking])
	actual = mem_work_ts.Get(zeroTime, zeroTime)
	assert.Len(actual, 1)
	assert.Equal(actual[0].Value.(uint64), uint64(2408))

	// Invocation with no fresh data - expect no change in cluster
	assert.NoError(cluster.Update(source_cache))
	verifyCacheFactoryCluster(&cluster.ClusterInfo, t)

	// Invocation with empty cache - expect no change in cluster
	assert.NoError(cluster.Update(empty_cache))
	verifyCacheFactoryCluster(&cluster.ClusterInfo, t)
}

// TestGetClusterMetric tests all flows of GetClusterMetric.
func TestGetClusterMetric(t *testing.T) {
	var (
		cluster      = newRealCluster(newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		zeroTime     = time.Time{}
	)
	// Invocation with no cluster metrics
	res, stamp, err := cluster.GetClusterMetric(ClusterRequest{cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant metric
	res, stamp, err = cluster.GetClusterMetric(ClusterRequest{"doesnotexist", zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	res, stamp, err = cluster.GetClusterMetric(ClusterRequest{memUsage, zeroTime, zeroTime})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetNodeMetric tests all flows of GetNodeMetric.
func TestGetNodeMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		hostname     = "hostname3"
	)
	// Invocation with no nodes in cluster
	res, stamp, err := cluster.GetNodeMetric(NodeRequest{hostname, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant node
	res, stamp, err = cluster.GetNodeMetric(NodeRequest{"doesnotexist", cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new node - no metrics
	cluster.addNode("newnode")
	res, stamp, err = cluster.GetNodeMetric(NodeRequest{"newnode", cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	res, stamp, err = cluster.GetNodeMetric(NodeRequest{hostname, "doesnotexist", zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	res, stamp, err = cluster.GetNodeMetric(NodeRequest{hostname, memUsage, zeroTime, zeroTime})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetNamespaceMetric tests all flows of GetNamespaceMetric.
func TestGetNamespaceMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		namespace    = "test"
	)
	// Invocation with no namespaces in cluster
	res, stamp, err := cluster.GetNamespaceMetric(NamespaceRequest{namespace, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant namespace
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceRequest{"doesnotexist", cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new namespace - no metrics
	cluster.addNamespace("newnode")
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceRequest{"newnode", cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceRequest{namespace, "doesnotexist", zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	res, stamp, err = cluster.GetNamespaceMetric(NamespaceRequest{namespace, memUsage, zeroTime, zeroTime})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetPodMetric tests all flows of GetPodMetric.
func TestGetPodMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		namespace    = "test"
		pod          = "pod1"
		node         = "hostname2"
	)
	// Invocation with no namespaces in cluster
	res, stamp, err := cluster.GetPodMetric(PodRequest{namespace, pod, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant namespace
	res, stamp, err = cluster.GetPodMetric(PodRequest{"doesnotexist", pod, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant pod
	res, stamp, err = cluster.GetPodMetric(PodRequest{namespace, "otherpod", cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new pod - no metrics
	cluster.addPod(pod, "1234", cluster.Namespaces[namespace], cluster.Nodes[node])
	res, stamp, err = cluster.GetPodMetric(PodRequest{namespace, pod, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	res, stamp, err = cluster.GetPodMetric(PodRequest{namespace, pod, "doesnotexist", zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	res, stamp, err = cluster.GetPodMetric(PodRequest{namespace, pod, memUsage, zeroTime, zeroTime})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetPodContainerMetric tests all flows of GetPodContainerMetric.
func TestGetPodContainerMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		namespace    = "test"
		pod          = "pod1"
		container    = "container1"
	)
	// Invocation with no namespaces in cluster
	res, stamp, err := cluster.GetPodContainerMetric(PodContainerRequest{namespace, pod, container, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant namespace
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerRequest{"doesnotexist", pod, container, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new namespace - no metrics
	cluster.addNamespace("newnode")
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerRequest{"newnode", pod, container, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant pod
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerRequest{namespace, "otherpod", container, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant container
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerRequest{namespace, pod, "doesnotexist", cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerRequest{namespace, pod, container, "doesnotexist", zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	res, stamp, err = cluster.GetPodContainerMetric(PodContainerRequest{namespace, pod, container, memUsage, zeroTime, zeroTime})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

// TestGetFreeContainerMetric tests all flows of GetFreeContainerMetric.
func TestGetFreeContainerMetric(t *testing.T) {
	var (
		zeroTime     = time.Time{}
		cluster      = newRealCluster(newTimeStore, time.Minute)
		source_cache = cacheFactory()
		assert       = assert.New(t)
		node         = "hostname2"
		container    = "free_container1"
	)
	// Invocation with no nodes in cluster
	res, stamp, err := cluster.GetFreeContainerMetric(FreeContainerRequest{node, container, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Populate cluster
	assert.NoError(cluster.Update(source_cache))

	// Invocation with non-existant node
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerRequest{"doesnotexist", container, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with new node - no metrics
	cluster.addNode("newnode")
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerRequest{"newnode", container, cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant container
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerRequest{node, "not_actual_ctr", cpuUsage, zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Invocation with non-existant metric
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerRequest{node, container, "doesnotexist", zeroTime, zeroTime})
	assert.Error(err)
	assert.Equal(stamp, zeroTime)
	assert.Nil(res)

	// Normal Invocation - memoryUsage
	res, stamp, err = cluster.GetFreeContainerMetric(FreeContainerRequest{node, container, memUsage, zeroTime, zeroTime})
	assert.NoError(err)
	assert.NotEqual(stamp, zeroTime)
	assert.NotNil(res)
}

//TestGetNodes tests the flow of GetNodes.
func TestGetNodes(t *testing.T) {
	var (
		cluster      = newRealCluster(newTimeStore, time.Minute)
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
		cluster      = newRealCluster(newTimeStore, time.Minute)
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
		cluster      = newRealCluster(newTimeStore, time.Minute)
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
		cluster      = newRealCluster(newTimeStore, time.Minute)
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
		cluster      = newRealCluster(newTimeStore, time.Minute)
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
		cluster      = newRealCluster(newTimeStore, time.Minute)
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

// verifyCacheFactoryCluster performs assertions over a ClusterInfo structure,
// based on the values and structure generated by cacheFactory.
func verifyCacheFactoryCluster(clinfo *ClusterInfo, t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(clinfo.Nodes["hostname2"])
	node2 := clinfo.Nodes["hostname2"]
	assert.NotEmpty(node2.Metrics)
	assert.Len(node2.FreeContainers, 1)
	assert.NotNil(node2.FreeContainers["free_container1"])

	assert.NotNil(clinfo.Nodes["hostname3"])
	node3 := clinfo.Nodes["hostname3"]
	assert.NotEmpty(node3.Metrics)

	assert.NotNil(clinfo.Namespaces["test"])
	namespace := clinfo.Namespaces["test"]

	assert.NotNil(namespace.Pods)
	pod1_ptr := namespace.Pods["pod1"]
	require.NotNil(t, pod1_ptr)
	assert.Equal(pod1_ptr, node2.Pods["pod1"])
	assert.Len(pod1_ptr.Containers, 2)
	pod2_ptr := namespace.Pods["pod2"]
	require.NotNil(t, pod2_ptr)
	assert.Equal(pod2_ptr, node3.Pods["pod2"])
	assert.Len(pod2_ptr.Containers, 2)
}

// Factory Functions

// cmeFactory generates a complete ContainerMetricElement with fuzzed data.
// CMEs created by cmeFactory contain partially fuzzed stats, aside from hardcoded values for Memory usage.
// The timestamp of the CME is rouded to the current minute and offset by a random number of hours.
func cmeFactory() *cache.ContainerMetricElement {
	f := fuzz.New().NilChance(0).NumElements(1, 1)
	containerSpec := cadvisor.ContainerSpec{
		CreationTime:  time.Now(),
		HasCpu:        true,
		HasMemory:     true,
		HasNetwork:    true,
		HasFilesystem: true,
		HasDiskIo:     true,
	}
	containerSpec.Cpu.Limit = 1000
	containerSpec.Memory.Limit = 10000000

	// Create a fuzzed ContainerStats struct
	var containerStats cadvisor.ContainerStats
	f.Fuzz(&containerStats)

	// Standardize timestamp to the current minute plus a random number of hours ([1, 10])
	now_time := time.Now().Round(time.Minute)
	new_time := now_time
	for new_time == now_time {
		new_time = now_time.Add(time.Duration(rand.Intn(10)) * time.Hour)
	}
	containerStats.Timestamp = new_time

	// Standardize memory usage and limit to test aggregation
	containerStats.Memory.Usage = uint64(5000)
	containerStats.Memory.WorkingSet = uint64(602)

	// Standardize the device name
	new_fs := cadvisor.FsStats{}
	f.Fuzz(&new_fs)
	new_fs.Device = "/dev/device1"
	containerStats.Filesystem = []cadvisor.FsStats{new_fs}

	return &cache.ContainerMetricElement{
		Spec:  &containerSpec,
		Stats: &containerStats,
	}
}

// emptyCMEFactory generates an empty ContainerMetricElement.
func emptyCMEFactory() *cache.ContainerMetricElement {
	f := fuzz.New().NilChance(0).NumElements(1, 1)
	containerSpec := cadvisor.ContainerSpec{
		CreationTime:  time.Now(),
		HasCpu:        false,
		HasMemory:     false,
		HasNetwork:    false,
		HasFilesystem: false,
		HasDiskIo:     false,
	}
	var containerStats cadvisor.ContainerStats
	f.Fuzz(&containerStats)
	containerStats.Timestamp = time.Now()

	return &cache.ContainerMetricElement{
		Spec:  &containerSpec,
		Stats: &containerStats,
	}
}

// containerElementFactory generates a new ContainerElement.
// The `cmes` argument represents a []*ContainerMetricElement, used for the Metrics field.
// If the `cmes` argument is nil, two ContainerMetricElements are automatically generated.
func containerElementFactory(cmes []*cache.ContainerMetricElement) *cache.ContainerElement {
	var metrics []*cache.ContainerMetricElement
	if cmes == nil {
		// If the argument is nil, generate two CMEs
		cme_1 := cmeFactory()
		cme_2 := cmeFactory()
		if !cme_1.Stats.Timestamp.After(cme_2.Stats.Timestamp) {
			// Ensure random but different timestamps
			cme_2.Stats.Timestamp = cme_1.Stats.Timestamp.Add(-2 * time.Minute)
		}
		metrics = []*cache.ContainerMetricElement{cme_1, cme_2}
	} else {
		metrics = cmes
	}
	metadata := cache.Metadata{
		Name:      "test",
		Namespace: "default",
		UID:       "123123123",
		Hostname:  "testhost",
		Labels:    make(map[string]string),
	}
	new_ce := cache.ContainerElement{
		Metadata: metadata,
		Metrics:  metrics,
	}
	return &new_ce
}

// podElementFactory creates a new PodElement with predetermined structure and fuzzed CME values.
// The resulting PodElement contains one ContainerElement with two ContainerMetricElements
func podElementFactory() *cache.PodElement {
	container_metadata := cache.Metadata{
		Name:      "test",
		Namespace: "default",
		UID:       "123123123",
		Hostname:  "testhost",
		Labels:    make(map[string]string),
	}
	cme_1 := cmeFactory()
	cme_2 := cmeFactory()
	for cme_1.Stats.Timestamp == cme_2.Stats.Timestamp {
		// Ensure random but different timestamps
		cme_2 = cmeFactory()
	}
	metrics := []*cache.ContainerMetricElement{cme_1, cme_2}

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

// cacheFactory generates a cache with a predetermined structure.
// The cache contains two pods, one with two containers and one without any containers.
// The cache also contains a free container and a "machine"-tagged container.
func cacheFactory() cache.Cache {
	source_cache := cache.NewCache(24 * time.Hour)

	// Generate Container CMEs - same timestamp for aggregation
	cme_1 := cmeFactory()
	cme_2 := cmeFactory()
	cme_2.Stats.Timestamp = cme_1.Stats.Timestamp

	// Genete Machine CMEs - same timestamp for aggregation
	cme_3 := cmeFactory()
	cme_4 := cmeFactory()
	cme_4.Stats.Timestamp = cme_3.Stats.Timestamp

	cme_5 := cmeFactory()
	cme_5.Stats.Timestamp = cme_4.Stats.Timestamp.Add(time.Hour)
	cme_5.Stats.Cpu.Usage.Total = cme_4.Stats.Cpu.Usage.Total + uint64(3600000000000)

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
			Containers: containers,
		},
	}

	// Generate two machine containers
	machine_container := source_api.Container{
		Name:     "/",
		Hostname: "hostname2",
		Spec:     *cme_3.Spec,
		Stats:    []*cadvisor.ContainerStats{cme_3.Stats},
	}
	machine_container2 := source_api.Container{
		Name:     "/",
		Hostname: "hostname3",
		Spec:     *cme_4.Spec,
		Stats:    []*cadvisor.ContainerStats{cme_5.Stats, cme_4.Stats},
	}
	// Generate a free container
	free_container := source_api.Container{
		Name:     "free_container1",
		Hostname: "hostname2",
		Spec:     *cme_5.Spec,
		Stats:    []*cadvisor.ContainerStats{cme_5.Stats},
	}

	other_containers := []source_api.Container{
		machine_container,
		machine_container2,
		free_container,
	}

	// Enter everything in the cache
	source_cache.StorePods(pods)
	source_cache.StoreContainers(other_containers)

	return source_cache
}
