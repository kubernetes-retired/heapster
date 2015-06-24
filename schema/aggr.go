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

	"github.com/GoogleCloudPlatform/heapster/store"
)

// aggregationStep performs a Metric Aggregation step on the cluster.
// The Metrics fields of all Namespaces, Pods and the Cluster are populated,
// by Timeseries summation of the respective Metrics fields.
// aggregationStep should be called after new data is present in the cluster,
// but before the cluster timestamp is updated.
func (rc *realCluster) aggregationStep() error {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Perform Node Metric Aggregation
	node_c := make(chan error)
	go rc.aggregateNodeMetrics(node_c)

	// Initiate bottom-up aggregation for Kubernetes stats
	kube_c := make(chan error)
	go rc.aggregateKubeMetrics(kube_c)

	err := <-node_c
	if err != nil {
		return err
	}
	err = <-kube_c
	if err != nil {
		return err
	}

	return nil
}

// aggregateNodeMetrics populates the Cluster.InfoType.Metrics field by adding up all node metrics.
// Assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) aggregateNodeMetrics(c chan error) {
	if len(rc.Nodes) == 0 {
		// Fail silently if the cluster has no nodes
		c <- nil
		return
	}

	sources := []*InfoType{}
	for _, node := range rc.Nodes {
		sources = append(sources, &(node.InfoType))
	}
	c <- rc.aggregateMetrics(&rc.ClusterInfo.InfoType, sources)
}

// aggregateKubeMetrics initiates depth-first aggregation of Kubernetes metrics.
// Assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) aggregateKubeMetrics(c chan error) {
	if len(rc.Namespaces) == 0 {
		// Fail silently if the cluster has no namespaces
		c <- nil
		return
	}

	// Perform aggregation for all the namespaces
	chans := make([]chan error, 0)
	for _, namespace := range rc.Namespaces {
		chans = append(chans, make(chan error))
		go rc.aggregateNamespaceMetrics(namespace, chans[len(chans)-1])
	}

	for _, channel := range chans {
		err := <-channel
		if err != nil {
			c <- err
		}
	}
	c <- nil
}

// aggregateNamespaceMetrics populates a NamespaceInfo.Metrics field by aggregating all PodInfo.
// Assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) aggregateNamespaceMetrics(namespace *NamespaceInfo, c chan error) {
	if namespace == nil {
		c <- fmt.Errorf("nil Namespace pointer passed for aggregation")
		return
	}
	if len(namespace.Pods) == 0 {
		// Fail silently if the namespace has no pods
		c <- nil
		return
	}

	// Perform aggregation for all the Pods
	chans := make([]chan error, 0)
	for _, pod := range namespace.Pods {
		chans = append(chans, make(chan error))
		go rc.aggregatePodMetrics(pod, chans[len(chans)-1])
	}

	for _, channel := range chans {
		err := <-channel
		if err != nil {
			c <- err
			return
		}
	}

	// Collect the Pod InfoTypes after aggregation is complete
	sources := []*InfoType{}
	for _, pod := range namespace.Pods {
		sources = append(sources, &(pod.InfoType))
	}
	c <- rc.aggregateMetrics(&namespace.InfoType, sources)
}

// aggregatePodMetrics populates a PodInfo.Metrics field by aggregating all ContainerInfo.
// Assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) aggregatePodMetrics(pod *PodInfo, c chan error) {
	if pod == nil {
		c <- fmt.Errorf("nil Pod pointer passed for aggregation")
		return
	}
	if len(pod.Containers) == 0 {
		// Fail silently if the pod has no containers
		c <- nil
		return
	}

	// Collect the Container InfoTypes
	sources := []*InfoType{}
	for _, container := range pod.Containers {
		sources = append(sources, &(container.InfoType))
	}
	c <- rc.aggregateMetrics(&pod.InfoType, sources)
}

// aggregateMetrics populates an InfoType by adding metrics across a slice of InfoTypes.
// Only metrics taken after the cluster timestamp are affected.
// Assumes an appropriate lock is already taken by the caller.
func (rc *realCluster) aggregateMetrics(target *InfoType, sources []*InfoType) error {
	zeroTime := time.Time{}

	if target == nil {
		return fmt.Errorf("nil InfoType pointer provided as aggregation target")
	}
	if len(sources) == 0 {
		return fmt.Errorf("empty sources slice provided")
	}
	for _, source := range sources {
		if source == nil {
			return fmt.Errorf("nil InfoType pointer provided as an aggregation source")
		}
		if source == target {
			return fmt.Errorf("target InfoType pointer is provided as a source")
		}
	}

	// Create a map of []TimePoint as a timeseries accumulator per metric
	newMetrics := make(map[string][]store.TimePoint)

	// Reduce the sources slice with timeseries addition for each metric
	for _, info := range sources {
		for key, ts := range info.Metrics {
			_, ok := newMetrics[key]
			if !ok {
				// Metric does not exist on target map, create a new timeseries
				newMetrics[key] = []store.TimePoint{}
			}
			// Perform timeseries addition between the accumulator and the current source
			sourceTS := (*ts).Get(rc.timestamp, zeroTime)
			newMetrics[key] = addMatchingTimeseries(newMetrics[key], sourceTS)
		}
	}

	// Put all the new values in the TimeStores under target
	for key, tpSlice := range newMetrics {
		_, ok := target.Metrics[key]
		if !ok {
			// Metric does not exist on target InfoType, create TimeStore
			newTS := rc.tsConstructor()
			target.Metrics[key] = &newTS
		}
		for _, tp := range tpSlice {
			(*target.Metrics[key]).Put(tp)
		}
	}
	return nil
}
