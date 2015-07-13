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
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	"github.com/GoogleCloudPlatform/heapster/store"
)

// NewCluster returns a new Cluster.
// Receives a TimeStore constructor function and a Duration resolution for stored data.
func NewCluster(tsConstructor func() store.TimeStore, resolution time.Duration) Cluster {
	return newRealCluster(tsConstructor, resolution)
}

// newRealCluster returns a realCluster, given a TimeStore constructor and a Duration resolution.
func newRealCluster(tsConstructor func() store.TimeStore, resolution time.Duration) *realCluster {
	cinfo := ClusterInfo{
		InfoType:   newInfoType(nil, nil, nil),
		Namespaces: make(map[string]*NamespaceInfo),
		Nodes:      make(map[string]*NodeInfo),
	}
	cluster := &realCluster{
		timestamp:     time.Time{},
		ClusterInfo:   cinfo,
		tsConstructor: tsConstructor,
		resolution:    resolution,
	}
	return cluster
}

// GetClusterMetric returns a metric of the cluster entity, along with the latest timestamp.
// GetClusterMetric receives as arguments the name of a metric and a starting timestamp.
// GetClusterMetric returns a slice of TimePoints for that metric, with times starting AFTER the starting timestamp.
func (rc *realCluster) GetClusterMetric(metric_name string, start time.Time) ([]store.TimePoint, time.Time, error) {
	var zeroTime time.Time
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	if len(rc.Metrics) == 0 {
		return nil, zeroTime, fmt.Errorf("cluster metrics are not populated yet")
	}

	ts, ok := rc.Metrics[metric_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested metric is not present")
	}
	res := (*ts).Get(start, zeroTime)
	return res, rc.timestamp, nil
}

// GetNodeMetric returns a metric of a node entity, along with the latest timestamp.
// GetNodeMetric receives as arguments the name of a metric and a starting timestamp.
// GetNodeMetric returns a slice of TimePoints for that metric, with times starting AFTER the starting timestamp.
func (rc *realCluster) GetNodeMetric(hostname string, metric_name string, start time.Time) ([]store.TimePoint, time.Time, error) {
	var zeroTime time.Time
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	if len(rc.Nodes) == 0 {
		return nil, zeroTime, fmt.Errorf("the model is not populated yet")
	}
	if _, ok := rc.Nodes[hostname]; !ok {
		return nil, zeroTime, fmt.Errorf("the requested node is not present in the cluster")
	}
	if len(rc.Nodes[hostname].Metrics) == 0 {
		return nil, zeroTime, fmt.Errorf("the requested node is not populated with metrics yet")
	}
	ts, ok := rc.Nodes[hostname].Metrics[metric_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested node metric is not present in the model")
	}

	res := (*ts).Get(start, zeroTime)
	return res, rc.timestamp, nil
}

// GetNamespaceMetric returns a metric of a namespace entity, along with the latest timestamp.
// GetNamespaceMetric receives as arguments the namespace, the metric name and a start time.
// GetNamespaceMetric returns a slice of TimePoints for that metric, with times starting AFTER the starting timestamp.
func (rc *realCluster) GetNamespaceMetric(namespace string, metric_name string, start time.Time) ([]store.TimePoint, time.Time, error) {
	var zeroTime time.Time
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	if len(rc.Namespaces) == 0 {
		return nil, zeroTime, fmt.Errorf("the model is not populated yet")
	}
	if _, ok := rc.Namespaces[namespace]; !ok {
		return nil, zeroTime, fmt.Errorf("the requested namespace is not present in the cluster")
	}
	if len(rc.Namespaces[namespace].Metrics) == 0 {
		return nil, zeroTime, fmt.Errorf("the requested namespace is not populated with metrics yet")
	}
	ts, ok := rc.Namespaces[namespace].Metrics[metric_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested namespace metric is not present in the model")
	}

	res := (*ts).Get(start, zeroTime)
	return res, rc.timestamp, nil
}

// GetPodMetric returns a metric of a Pod entity, along with the latest timestamp.
// GetPodMetric receives as arguments the namespace, the pod name, the metric name and a start time.
// GetPodMetric returns a slice of TimePoints for that metric, with times starting AFTER the starting timestamp.
func (rc *realCluster) GetPodMetric(namespace string, pod_name string, metric_name string, start time.Time) ([]store.TimePoint, time.Time, error) {
	var zeroTime time.Time
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	if len(rc.Namespaces) == 0 {
		return nil, zeroTime, fmt.Errorf("the model is not populated yet")
	}
	ns, ok := rc.Namespaces[namespace]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the specified namespace is not present in the cluster")
	}
	pod, ok := ns.Pods[pod_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested pod is not present in the specified namespace")
	}
	if len(pod.Metrics) == 0 {
		return nil, zeroTime, fmt.Errorf("the requested pod is not populated with metrics yet")
	}
	ts, ok := pod.Metrics[metric_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested pod metric is not present in the model")
	}

	res := (*ts).Get(start, zeroTime)
	return res, rc.timestamp, nil
}

// GetPodContainerMetric returns a metric of a container entity that belongs in a Pod, along with the latest timestamp.
// GetPodContainerMetric receives as arguments the namespace, the pod name, the container name, the metric name and a start time.
// GetPodContainerMetric returns a slice of TimePoints for that metric, with times starting AFTER the starting timestamp.
func (rc *realCluster) GetPodContainerMetric(namespace string, pod_name string, container_name string, metric_name string, start time.Time) ([]store.TimePoint, time.Time, error) {
	var zeroTime time.Time
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	if len(rc.Namespaces) == 0 {
		return nil, zeroTime, fmt.Errorf("the model is not populated yet")
	}
	ns, ok := rc.Namespaces[namespace]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the specified namespace is not present in the cluster")
	}
	pod, ok := ns.Pods[pod_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the specified pod is not present in the specified namespace")
	}
	ctr, ok := pod.Containers[container_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested container is not present under the specified pod")
	}
	ts, ok := ctr.Metrics[metric_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested container metric is not present in the model")
	}

	res := (*ts).Get(start, zeroTime)
	return res, rc.timestamp, nil
}

// GetFreeContainerMetric returns a metric of a free container entity, along with the latest timestamp.
// GetFreeContainerMetric receives as arguments the host name, the container name, the metric name and a start time.
// GetFreeContainerMetric returns a slice of TimePoints for that metric, with times starting AFTER the starting timestamp.
func (rc *realCluster) GetFreeContainerMetric(hostname string, container_name string, metric_name string, start time.Time) ([]store.TimePoint, time.Time, error) {
	var zeroTime time.Time
	rc.lock.RLock()
	defer rc.lock.RUnlock()
	if len(rc.Nodes) == 0 {
		return nil, zeroTime, fmt.Errorf("the model is not populated yet")
	}
	node, ok := rc.Nodes[hostname]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested node is not present in the cluster")
	}
	ctr, ok := node.FreeContainers[container_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested container is not present under the specified node")
	}
	ts, ok := ctr.Metrics[metric_name]
	if !ok {
		return nil, zeroTime, fmt.Errorf("the requested container metric is not present in the model")
	}

	res := (*ts).Get(start, zeroTime)
	return res, rc.timestamp, nil
}

// GetAvailableMetrics returns the names of all metrics that are available on the cluster.
// Due to metric propagation, all entities of the cluster have the same metrics.
func (rc *realCluster) GetAvailableMetrics() []string {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	res := make([]string, 0)
	for key, _ := range rc.Metrics {
		res = append(res, key)
	}
	return res
}

// updateTime updates the Cluster timestamp to the specified time.
func (rc *realCluster) updateTime(new_time time.Time) {
	if new_time.Equal(time.Time{}) {
		return
	}
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
			InfoType:       newInfoType(nil, nil, nil),
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
			InfoType: newInfoType(nil, nil, nil),
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

	if namespace == nil {
		glog.V(2).Infof("nil namespace pointer passed to addPod")
		return nil
	}

	if node == nil {
		glog.V(2).Infof("nil node pointer passed to addPod")
		return nil
	}

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
			InfoType:   newInfoType(nil, nil, nil),
			UID:        pod_uid,
			Containers: make(map[string]*ContainerInfo),
		}
		namespace.Pods[pod_name] = pod_ptr
		node.Pods[pod_name] = pod_ptr
	}

	return pod_ptr
}

// updateInfoType updates the metrics of an InfoType from a ContainerElement.
// updateInfoType returns the latest timestamp in the resulting TimeStore.
// updateInfoType does not fail if a single ContainerMetricElement cannot be parsed.
func (rc *realCluster) updateInfoType(info *InfoType, ce *cache.ContainerElement) (time.Time, error) {
	var latest_time time.Time

	if ce == nil {
		return latest_time, fmt.Errorf("cannot update InfoType from nil ContainerElement")
	}
	if info == nil {
		return latest_time, fmt.Errorf("cannot update a nil InfoType")
	}

	for _, cme := range ce.Metrics {
		stamp, err := rc.parseMetric(cme, info.Metrics, info.Context)
		if err != nil {
			glog.V(2).Infof("failed to parse ContainerMetricElement: %s", err)
			continue
		}
		latest_time = latestTimestamp(latest_time, stamp)
	}
	return latest_time, nil
}

// addMetricToMap adds a new metric (time-value pair) to a map of TimeStores.
// addMetricToMap accepts as arguments the metric name, timestamp, value and the TimeStore map.
// The timestamp argument needs to be already rounded to the cluster resolution.
func (rc *realCluster) addMetricToMap(metric string, timestamp time.Time, value uint64, dict map[string]*store.TimeStore) error {
	point := store.TimePoint{
		Timestamp: timestamp,
		Value:     value,
	}
	if val, ok := dict[metric]; ok {
		ts := *val
		err := ts.Put(point)
		if err != nil {
			return fmt.Errorf("failed to add metric to TimeStore: %s", err)
		}
	} else {
		new_ts := rc.tsConstructor()
		err := new_ts.Put(point)
		if err != nil {
			return fmt.Errorf("failed to add metric to TimeStore: %s", err)
		}
		dict[metric] = &new_ts
	}
	return nil
}

// parseMetric populates a map[string]*TimeStore from a ContainerMetricElement.
// parseMetric returns the ContainerMetricElement timestamp, iff successful.
func (rc *realCluster) parseMetric(cme *cache.ContainerMetricElement, dict map[string]*store.TimeStore, context map[string]*store.TimePoint) (time.Time, error) {
	zeroTime := time.Time{}
	if cme == nil {
		return zeroTime, fmt.Errorf("cannot parse nil ContainerMetricElement")
	}
	if dict == nil {
		return zeroTime, fmt.Errorf("cannot populate nil map")
	}
	if context == nil {
		return zeroTime, fmt.Errorf("nil context provided to parseMetric")
	}

	// Round the timestamp to the nearest resolution
	timestamp := cme.Stats.Timestamp
	roundedStamp := timestamp.Round(rc.resolution)

	// TODO(alex): refactor to avoid repetition
	if cme.Spec.HasCpu {
		// Append to CPU Limit metric
		cpu_limit := cme.Spec.Cpu.Limit
		err := rc.addMetricToMap(cpuLimit, roundedStamp, cpu_limit, dict)
		if err != nil {
			return zeroTime, fmt.Errorf("failed to add %s metric: %s", cpuLimit, err)
		}

		// Get the new cumulative CPU Usage datapoint
		cpu_usage := cme.Stats.Cpu.Usage.Total

		// use the context to store a TimePoint of the previous cumulative cpuUsage.
		prevTP, ok := context[cpuUsage]
		if !ok && cpu_usage != 0 {
			// Context is empty, add the first TimePoint for cumulative cpuUsage.
			context[cpuUsage] = &store.TimePoint{
				Timestamp: timestamp,
				Value:     cpu_usage,
			}
		} else if !roundedStamp.Equal(prevTP.Timestamp.Round(rc.resolution)) {
			// Context is not empty and the new CPU Usage does not round to the same value
			// Calculate new instantaneous CPU Usage
			newCPU, err := instantFromCumulativeMetric(cpu_usage, timestamp, prevTP)
			if err != nil {
				return zeroTime, fmt.Errorf("failed to calculate instantaneous CPU usage: %s", err)
			}
			context[cpuUsage] = prevTP

			// Add to CPU Usage metric
			err = rc.addMetricToMap(cpuUsage, roundedStamp, newCPU, dict)
			if err != nil {
				return zeroTime, fmt.Errorf("failed to add %s metric: %s", cpuUsage, err)
			}
		}
	}

	if cme.Spec.HasMemory {
		// Add Memory Limit metric
		mem_limit := cme.Spec.Memory.Limit
		err := rc.addMetricToMap(memLimit, roundedStamp, mem_limit, dict)
		if err != nil {
			return zeroTime, fmt.Errorf("failed to add %s metric: %s", memLimit, err)
		}

		// Add Memory Usage metric
		mem_usage := cme.Stats.Memory.Usage
		err = rc.addMetricToMap(memUsage, roundedStamp, mem_usage, dict)
		if err != nil {
			return zeroTime, fmt.Errorf("failed to add %s metric: %s", memUsage, err)
		}

		// Add Memory Working Set metric
		mem_working := cme.Stats.Memory.WorkingSet
		err = rc.addMetricToMap(memWorking, roundedStamp, mem_working, dict)
		if err != nil {
			return zeroTime, fmt.Errorf("failed to add %s metric: %s", memWorking, err)
		}
	}
	if cme.Spec.HasFilesystem {
		for _, fsstat := range cme.Stats.Filesystem {
			dev := fsstat.Device

			// Add FS Limit Metric
			fs_limit := fsstat.Limit
			metric_name := fsLimit + strings.Replace(dev, "/", "-", -1)
			err := rc.addMetricToMap(metric_name, roundedStamp, fs_limit, dict)
			if err != nil {
				return zeroTime, fmt.Errorf("failed to add %s metric: %s", fsLimit, err)
			}

			// Add FS Usage Metric
			fs_usage := fsstat.Usage
			metric_name = fsUsage + strings.Replace(dev, "/", "-", -1)
			err = rc.addMetricToMap(metric_name, roundedStamp, fs_usage, dict)
			if err != nil {
				return zeroTime, fmt.Errorf("failed to add %s metric: %s", fsUsage, err)
			}
		}
	}
	return roundedStamp, nil
}

// Update populates the data structure from a cache.
func (rc *realCluster) Update(c cache.Cache) error {
	var zero time.Time
	latest_time := rc.timestamp
	glog.V(2).Infoln("Schema Update operation started")

	// Invoke cache methods using the Cluster timestamp
	nodes := c.GetNodes(rc.timestamp, zero)
	for _, node := range nodes {
		timestamp, err := rc.updateNode(node)
		if err != nil {
			return fmt.Errorf("Failed to Update Node Information: %s", err)
		}
		latest_time = latestTimestamp(latest_time, timestamp)
	}

	pods := c.GetPods(rc.timestamp, zero)
	for _, pod := range pods {
		timestamp, err := rc.updatePod(pod)
		if err != nil {
			return fmt.Errorf("Failed to Update Pod Information: %s", err)
		}
		latest_time = latestTimestamp(latest_time, timestamp)
	}

	free_containers := c.GetFreeContainers(rc.timestamp, zero)
	for _, ce := range free_containers {
		timestamp, err := rc.updateFreeContainer(ce)
		if err != nil {
			return fmt.Errorf("Failed to Update Free Container Information: %s", err)
		}
		latest_time = latestTimestamp(latest_time, timestamp)
	}

	// Perform metrics aggregation
	rc.aggregationStep()

	// Update the Cluster timestamp to the latest time found in the new metrics
	rc.updateTime(latest_time)

	glog.V(2).Infoln("Schema Update operation completed")
	return nil
}

// updateNode updates Node-level information from a "machine"-tagged ContainerElement.
func (rc *realCluster) updateNode(node_container *cache.ContainerElement) (time.Time, error) {
	if node_container.Name != "machine" {
		return time.Time{}, fmt.Errorf("Received node-level container with unexpected name: %s", node_container.Name)
	}

	rc.lock.Lock()
	defer rc.lock.Unlock()
	node_ptr := rc.addNode(node_container.Hostname)

	// Update NodeInfo's Metrics and Labels - return latest metric timestamp
	result, err := rc.updateInfoType(&node_ptr.InfoType, node_container)
	return result, err
}

// updatePod updates Pod-level information from a PodElement.
func (rc *realCluster) updatePod(pod *cache.PodElement) (time.Time, error) {
	if pod == nil {
		return time.Time{}, fmt.Errorf("nil PodElement provided to updatePod")
	}

	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Get Namespace and Node pointers
	namespace := rc.addNamespace(pod.Namespace)
	node := rc.addNode(pod.Hostname)

	// Get Pod pointer
	pod_ptr := rc.addPod(pod.Name, pod.UID, namespace, node)

	// Copy Labels map
	pod_ptr.Labels = pod.Labels

	// Update container metrics
	latest_time := time.Time{}
	for _, ce := range pod.Containers {
		new_time, err := rc.updatePodContainer(pod_ptr, ce)
		if err != nil {
			return time.Time{}, err
		}
		latest_time = latestTimestamp(latest_time, new_time)
	}

	return latest_time, nil
}

// updatePodContainer updates a Pod's Container-level information from a ContainerElement.
// updatePodContainer receives a PodInfo pointer and a ContainerElement pointer.
// Assumes Cluster lock is already taken.
func (rc *realCluster) updatePodContainer(pod_info *PodInfo, ce *cache.ContainerElement) (time.Time, error) {
	// Get Container pointer and update its InfoType
	cinfo := addContainerToMap(ce.Name, pod_info.Containers)
	latest_time, err := rc.updateInfoType(&cinfo.InfoType, ce)
	return latest_time, err
}

// updateFreeContainer updates Free Container-level information from a ContainerElement
func (rc *realCluster) updateFreeContainer(ce *cache.ContainerElement) (time.Time, error) {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Get Node pointer
	node := rc.addNode(ce.Hostname)
	// Get Container pointer and update its InfoType
	cinfo := addContainerToMap(ce.Name, node.FreeContainers)
	latest_time, err := rc.updateInfoType(&cinfo.InfoType, ce)
	return latest_time, err
}
