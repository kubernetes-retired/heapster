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

package cache

import (
	"sync"
	"time"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
)

type containerElement struct {
	Metadata
	metrics TimeStore
}

type podElement struct {
	Metadata
	// map of container name to container element.
	containers map[string]*containerElement
	// TODO: Cache history of Spec and Status.
}

type nodeElement struct {
	node *containerElement
	// FreeContainers refers to all the containers in a node
	// that do not belong to a pod.
	freeContainers map[string]*containerElement
	// TODO: Cache history of Spec and Status.
}

type realCache struct {
	bufferDuration time.Duration
	// Map of pod UIDs to pod cache entry.
	pods map[string]*podElement
	// Map of node hostnames to node cache entry.
	nodes map[string]*nodeElement
	lock  sync.RWMutex
}

const rootContainer = "/"

func (rc *realCache) newContainerElement() *containerElement {
	return &containerElement{
		metrics: NewGCStore(NewTimeStore(), rc.bufferDuration),
	}
}

func (rc *realCache) newpodElement() *podElement {
	return &podElement{
		containers: make(map[string]*containerElement),
	}
}

func (rc *realCache) newnodeElement() *nodeElement {
	return &nodeElement{
		node:           rc.newContainerElement(),
		freeContainers: make(map[string]*containerElement),
	}
}

func storeSpecAndStats(ce *containerElement, c *source_api.Container) {
	if ce == nil || c == nil {
		return
	}
	for idx := range c.Stats {
		if c.Stats[idx] == nil {
			continue
		}
		cme := &ContainerMetricElement{
			Spec:  &c.Spec,
			Stats: c.Stats[idx],
		}
		ce.metrics.Put(c.Stats[idx].Timestamp, cme)
	}
}

func (rc *realCache) StorePods(pods []source_api.Pod) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	for _, pod := range pods {
		pe, ok := rc.pods[pod.ID]
		if !ok {
			pe = rc.newpodElement()
			pe.Metadata = Metadata{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				UID:       pod.ID,
				Hostname:  pod.Hostname,
				Labels:    pod.Labels,
			}
			rc.pods[pod.ID] = pe
		}
		for idx := range pod.Containers {
			cont := &pod.Containers[idx]
			ce, ok := pe.containers[cont.Name]
			if !ok {
				ce = rc.newContainerElement()
				pe.containers[cont.Name] = ce
			}
			ce.Metadata = Metadata{
				Name:     cont.Name,
				Hostname: cont.Hostname,
			}
			storeSpecAndStats(ce, cont)
		}
	}
	return nil
}

func (rc *realCache) StoreContainers(containers []source_api.Container) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	for idx := range containers {
		cont := &containers[idx]
		ne, ok := rc.nodes[cont.Hostname]
		if !ok {
			ne = rc.newnodeElement()
			rc.nodes[cont.Hostname] = ne
		}
		var ce *containerElement
		if cont.Name == rootContainer {
			// This is at the node level.
			ne.node.Hostname = cont.Hostname
			ne.node.Name = NodeContainerName
			ce = ne.node
		} else {
			var ok bool
			ce, ok = ne.freeContainers[cont.Name]
			if !ok {
				ce = rc.newContainerElement()
				ce.Metadata = Metadata{
					Name:     cont.Name,
					Hostname: cont.Hostname,
				}
				ne.freeContainers[cont.Name] = ce
			}
		}
		storeSpecAndStats(ce, cont)
	}
	return nil
}

func (rc *realCache) GetPods(start, end time.Time) []*PodElement {
	rc.lock.RLock()
	defer rc.lock.RUnlock()
	var result []*PodElement
	for _, pe := range rc.pods {
		podElement := &PodElement{
			Metadata: pe.Metadata,
		}
		for _, ce := range pe.containers {
			containerElement := &ContainerElement{
				Metadata: ce.Metadata,
			}
			metrics := ce.metrics.Get(start, end)
			for idx := range metrics {
				cme := metrics[idx].(*ContainerMetricElement)
				containerElement.Metrics = append(containerElement.Metrics, cme)
			}
			podElement.Containers = append(podElement.Containers, containerElement)
		}
		result = append(result, podElement)
	}

	return result
}

func (rc *realCache) GetNodes(start, end time.Time) []*ContainerElement {
	rc.lock.RLock()
	defer rc.lock.RUnlock()
	var result []*ContainerElement
	for _, ne := range rc.nodes {
		ce := &ContainerElement{
			Metadata: ne.node.Metadata,
		}
		metrics := ne.node.metrics.Get(start, end)
		for idx := range metrics {
			cme := metrics[idx].(*ContainerMetricElement)
			ce.Metrics = append(ce.Metrics, cme)
		}
		result = append(result, ce)
	}
	return result
}

func (rc *realCache) GetFreeContainers(start, end time.Time) []*ContainerElement {
	rc.lock.RLock()
	defer rc.lock.RUnlock()
	var result []*ContainerElement
	for _, ne := range rc.nodes {
		for _, ce := range ne.freeContainers {
			containerElement := &ContainerElement{
				Metadata: ce.Metadata,
			}
			metrics := ce.metrics.Get(start, end)
			for idx := range metrics {
				cme := metrics[idx].(*ContainerMetricElement)
				containerElement.Metrics = append(containerElement.Metrics, cme)
			}
			result = append(result, containerElement)
		}
	}
	return result
}

func NewCache(bufferDuration time.Duration) Cache {
	return &realCache{
		pods:           make(map[string]*podElement),
		nodes:          make(map[string]*nodeElement),
		bufferDuration: bufferDuration,
	}
}
