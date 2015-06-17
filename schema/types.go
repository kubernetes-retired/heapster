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
	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	"sync"
	"time"
)

type Cluster interface {
	// The Update operation populates the Cluster from a cache
	Update(*cache.Cache) error

	// The Get operations extract internal types from the Cluster
	GetAllClusterData() (*ClusterInfo, time.Time, error)
	GetAllNodeData(string) (*NodeInfo, time.Time, error)
	GetAllPodData(string, string) (*PodInfo, time.Time, error)
}

type realCluster struct {
	// Implementation of Cluster
	timestamp time.Time // Marks the last update from a cache.
	lock      *sync.RWMutex
	ClusterInfo
}

// Internal Types
// REST consumption requires conversion to the corresponding versioned API types

type InfoType struct {
	Metrics map[string]*cache.TimeStore // key: Metric Name
	Labels  map[string]string           // key: Label
}

type ClusterInfo struct {
	InfoType
	Namespaces map[string]*NamespaceInfo // key: Namespace Name
	Nodes      map[string]*NodeInfo      // key: Hostname
}

type NamespaceInfo struct {
	InfoType
	Pods map[string]*PodInfo // key: Pod Name
}

type NodeInfo struct {
	InfoType
	Pods           map[string]*PodInfo       // key: Pod Name
	FreeContainers map[string]*ContainerInfo // key: Container Name
}

type PodInfo struct {
	InfoType
	UID        string
	Containers map[string]*ContainerInfo // key: Container Name
}

type ContainerInfo struct {
	InfoType
}
