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
func NewCluster(dayConstructor func() store.DayStore, tsConstructor func() store.TimeStore, resolution time.Duration) Cluster {
	return newRealCluster(dayConstructor, tsConstructor, resolution)
}

// newRealCluster returns a realCluster, given a TimeStore constructor and a Duration resolution.
func newRealCluster(dayConstructor func() store.DayStore, tsConstructor func() store.TimeStore, resolution time.Duration) *realCluster {
	cinfo := ClusterInfo{
		InfoType:   newInfoType(nil, nil, nil),
		Namespaces: make(map[string]*NamespaceInfo),
		Nodes:      make(map[string]*NodeInfo),
	}
	cluster := &realCluster{
		timestamp:      time.Time{},
		ClusterInfo:    cinfo,
		dayConstructor: dayConstructor,
		tsConstructor:  tsConstructor,
		resolution:     resolution,
	}
	return cluster
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
	var latest_creation time.Time

	if ce == nil {
		return latest_time, fmt.Errorf("cannot update InfoType from nil ContainerElement")
	}
	if info == nil {
		return latest_time, fmt.Errorf("cannot update a nil InfoType")
	}

	for i := len(ce.Metrics) - 1; i >= 0; i-- {
		cme := ce.Metrics[i]
		if cme == nil {
			continue
		}
		latest_creation = latestTimestamp(latest_creation, cme.Spec.CreationTime)
		stamp, err := rc.parseMetric(cme, info.Metrics, info.Context)
		if err != nil {
			glog.Warningf("failed to parse ContainerMetricElement: %s", err)
			continue
		}
		latest_time = latestTimestamp(latest_time, stamp)
	}
	info.Uptime = latest_time.Sub(latest_creation)
	return latest_time, nil
}

// addMetricToMap adds a new metric (time-value pair) to a map of TimeStores.
// addMetricToMap accepts as arguments the metric name, timestamp, value and the TimeStore map.
// The timestamp argument needs to be already rounded to the cluster resolution.
func (rc *realCluster) addMetricToMap(metric string, timestamp time.Time, value uint64, dict map[string]*store.DayStore) error {
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
		new_ts := rc.dayConstructor()
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
func (rc *realCluster) parseMetric(cme *cache.ContainerMetricElement, dict map[string]*store.DayStore, context map[string]*store.TimePoint) (time.Time, error) {
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
	roundedStamp := timestamp.Truncate(rc.resolution)

	// TODO(alex): refactor to avoid repetition
	if cme.Spec.HasCpu {
		// Append to CPU Limit metric
		cpu_limit := cme.Spec.Cpu.Limit * 1000 / 1024 // convert to millicores
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
		} else {
			prevRoundedStamp := prevTP.Timestamp.Truncate(rc.resolution)

			if cme.Spec.CreationTime.After(prevTP.Timestamp) {
				// check if the container was restarted since the last context timestamp
				// TODO(afein): mark as a container crash event
				// Reset the context
				context[cpuUsage] = &store.TimePoint{
					Timestamp: timestamp,
					Value:     cpu_usage,
				}
			} else if prevRoundedStamp.Before(roundedStamp) {
				// Calculate new instantaneous CPU Usage
				newCPU, err := instantFromCumulativeMetric(cpu_usage, timestamp, prevTP)
				if err != nil {
					return zeroTime, fmt.Errorf("failed to calculate instantaneous CPU usage: %s", err)
				}

				// Add to CPU Usage metric
				err = rc.addMetricToMap(cpuUsage, roundedStamp, newCPU, dict)
				if err != nil {
					return zeroTime, fmt.Errorf("failed to add %s metric: %s", cpuUsage, err)
				}
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
	// Iterate through the results in time-ascending order to maintain the context for cumulative metrics

	nodes := c.GetNodes(rc.timestamp, zero)
	for i := len(nodes) - 1; i >= 0; i-- {
		timestamp, err := rc.updateNode(nodes[i])
		if err != nil {
			return fmt.Errorf("Failed to Update Node Information: %s", err)
		}
		latest_time = latestTimestamp(latest_time, timestamp)
	}

	pods := c.GetPods(rc.timestamp, zero)
	for i := len(pods) - 1; i >= 0; i-- {
		timestamp, err := rc.updatePod(pods[i])
		if err != nil {
			return fmt.Errorf("Failed to Update Pod Information: %s", err)
		}
		latest_time = latestTimestamp(latest_time, timestamp)
	}

	freeConts := c.GetFreeContainers(rc.timestamp, zero)
	for i := len(freeConts) - 1; i >= 0; i-- {
		timestamp, err := rc.updateFreeContainer(freeConts[i])
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
