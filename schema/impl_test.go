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
	"strings"
	"testing"
	"time"

	cadvisor "github.com/google/cadvisor/info/v1"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/store"
)

// newTimeStore creates a new GCStore and returns it as a TimeStore.
// Meant to be passed to newRealCluster calls in all unit tests.
func newTimeStore() store.TimeStore {
	return store.NewGCStore(store.NewTimeStore(), time.Hour)
}

// TestNewCluster tests the sanity of NewCluster
func TestNewCluster(t *testing.T) {
	cluster := NewCluster(newTimeStore)
	assert.NotNil(t, cluster)
}

// TestAddNamespace tests all flows of addNamespace.
func TestAddNamespace(t *testing.T) {
	var (
		cluster        = newRealCluster(newTimeStore)
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
		cluster  = newRealCluster(newTimeStore)
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
		cluster   = newRealCluster(newTimeStore)
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
		cluster = newRealCluster(newTimeStore)
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
		cluster                = newRealCluster(newTimeStore)
		metrics                = make(map[string]*store.TimeStore)
		new_metric_name string = "name_already_in_map"
		value           uint64 = 1234567890
		zeroTime               = time.Time{}
		stamp                  = time.Now()
		assert                 = assert.New(t)
	)

	// Put a dummy metric in the map
	new_tp := store.TimePoint{
		Timestamp: stamp,
		Value:     123,
	}
	ts := newTimeStore()
	assert.NoError(ts.Put(new_tp))
	metrics[new_metric_name] = &ts

	// Fist Call: addMetricToMap for a new metric with the same name
	assert.NoError(cluster.addMetricToMap(new_metric_name, stamp, value, metrics))

	new_ts := *metrics[new_metric_name]
	results := new_ts.Get(zeroTime, zeroTime)

	// Expect both metrics to be available through Get
	assert.Len(results, 2)
	assert.Equal(results[0].Timestamp, stamp)
	assert.Equal(results[0].Value, 123)
	assert.Equal(results[1].Timestamp, stamp)
	assert.Equal(results[1].Value, value)

	// Second Call: addMetricToMap for an existing key, cause TimeStore failure
	assert.Error(cluster.addMetricToMap(new_metric_name, zeroTime, value, metrics))
}

// Tests the flow of AddMetricToMap where the metric name is not present in the map
func TestAddMetricToMapNewKey(t *testing.T) {
	var (
		cluster                = newRealCluster(newTimeStore)
		metrics                = make(map[string]*store.TimeStore)
		new_metric_name        = "name_not_in_map"
		stamp                  = time.Now()
		zeroTime               = time.Time{}
		value           uint64 = 1234567890
		assert                 = assert.New(t)
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
		cluster = newRealCluster(newTimeStore)
		assert  = assert.New(t)
	)

	// Invoke parseMetric with a nil cme argument
	stamp, err := cluster.parseMetric(nil, make(map[string]*store.TimeStore))
	assert.Error(err)
	assert.Equal(stamp, time.Time{})

	// Invoke parseMetric with a nil dict argument
	cme := cmeFactory()
	stamp, err = cluster.parseMetric(cme, nil)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})
}

// TestParseMetricNormal tests the normal flow of ParseMetric
func TestParseMetricNormal(t *testing.T) {
	var (
		cluster    = newRealCluster(newTimeStore)
		zeroTime   = time.Time{}
		metrics    = make(map[string]*store.TimeStore)
		normal_cme = cmeFactory()
		assert     = assert.New(t)
	)

	// Normal Invocation with a regular CME
	stamp, err := cluster.parseMetric(normal_cme, metrics)
	assert.NoError(err)
	assert.Equal(stamp, normal_cme.Stats.Timestamp)
	for key, ts := range metrics {
		actual_ts := *ts
		pointSlice := actual_ts.Get(zeroTime, zeroTime)
		assert.Len(pointSlice, 1)
		metric := pointSlice[0]
		assert.Equal(metric.Timestamp, normal_cme.Stats.Timestamp)
		switch key {
		case "cpu/limit":
			assert.Equal(metric.Value, normal_cme.Spec.Cpu.Limit)
		case "memory/limit":
			assert.Equal(metric.Value, normal_cme.Spec.Memory.Limit)
		case "cpu/usage":
			assert.Equal(metric.Value, normal_cme.Stats.Cpu.Usage.Total)
		case "memory/usage":
			assert.Equal(metric.Value, normal_cme.Stats.Memory.Usage)
		case "memory/working":
			assert.Equal(metric.Value, normal_cme.Stats.Memory.WorkingSet)
		default:
			// Filesystem or error
			if strings.Contains(key, "limit") {
				assert.Equal(metric.Value, normal_cme.Stats.Filesystem[0].Limit)
			} else if strings.Contains(key, "usage") {
				assert.Equal(metric.Value, normal_cme.Stats.Filesystem[0].Usage)
			} else {
				assert.True(false, "Unknown key in resulting metrics slice")
			}
		}
	}
}

// TestUpdateInfoTypeError Tests the error flows of updateInfoType.
func TestUpdateInfoTypeError(t *testing.T) {
	var (
		cluster      = newRealCluster(newTimeStore)
		new_infotype = newInfoType(nil, nil)
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
		cluster      = newRealCluster(newTimeStore)
		new_cme      = cmeFactory()
		empty_ce     = containerElementFactory([]*cache.ContainerMetricElement{})
		nil_ce       = containerElementFactory([]*cache.ContainerMetricElement{new_cme, nil})
		new_ce       = containerElementFactory([]*cache.ContainerMetricElement{new_cme})
		new_infotype = newInfoType(nil, nil)
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
	assert.Len(new_infotype.Metrics, 7) // 7 stats in total
	for _, metricStore := range new_infotype.Metrics {
		metricSlice := (*metricStore).Get(zeroTime, zeroTime)
		assert.Len(metricSlice, 1) // 1 Metric per stat
	}

	// Invocation with an empty InfoType argument
	// The new ContainerElement adds one TimePoint to each of 7 Metrics
	stamp, err = cluster.updateInfoType(&new_infotype, new_ce)
	assert.NoError(err)
	assert.Empty(new_infotype.Labels)
	assert.NotEqual(stamp, time.Time{})
	assert.Len(new_infotype.Metrics, 7) // 7 stats in total
	for _, metricStore := range new_infotype.Metrics {
		metricSlice := (*metricStore).Get(zeroTime, zeroTime)
		assert.Len(metricSlice, 2) // 2 Metrics per stat
	}

	// Invocation with an existing infotype as argument
	// The new ContainerElement adds two TimePoints to each Metric
	new_ce = containerElementFactory(nil)
	stamp, err = cluster.updateInfoType(&new_infotype, new_ce)
	assert.NoError(err)
	assert.Empty(new_infotype.Labels)
	assert.NotEqual(stamp, time.Time{})
	assert.Len(new_infotype.Metrics, 7) // 7 stats total
	for _, metricStore := range new_infotype.Metrics {
		metricSlice := (*metricStore).Get(zeroTime, zeroTime)
		assert.Len(metricSlice, 4) // 4 Metrics per stat
	}
}

// TestUpdateFreeContainer tests the flow of updateFreeContainer
func TestUpdateFreeContainer(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore)
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
		cluster   = newRealCluster(newTimeStore)
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
		cluster  = newRealCluster(newTimeStore)
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
		cluster = newRealCluster(newTimeStore)
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
		cluster = newRealCluster(newTimeStore)
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
		cluster = newRealCluster(newTimeStore)
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

// TestUpdate tests the normal flow of Update.
func TestUpdate(t *testing.T) {
	var (
		cluster      = newRealCluster(newTimeStore)
		source_cache = cacheFactory()
	)
	// Invocation with regular parameters
	assert.NoError(t, cluster.Update(source_cache))
	verifyCacheFactoryCluster(&cluster.ClusterInfo, t)
}

// Factory Functions

// cmeFactory generates a complete ContainerMetricElement with fuzzed data.
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
	var containerStats cadvisor.ContainerStats
	f.Fuzz(&containerStats)
	containerStats.Timestamp = time.Now()

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
		metrics = []*cache.ContainerMetricElement{cmeFactory(), cmeFactory()}
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

// cacheFactory generates a cache with a predetermined structure.
// The cache contains two pods, one with two containers and one without any containers.
// The cache also contains a free container and a "machine"-tagged container.
func cacheFactory() cache.Cache {
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

	// Enter everything in the cache
	source_cache.StorePods(pods)
	source_cache.StoreContainers(other_containers)

	return source_cache
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
	assert.Empty(node3.Metrics)

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
