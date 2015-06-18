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
	"time"

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	"github.com/GoogleCloudPlatform/heapster/store"
)

type Cluster interface {
	// The Update operation populates the Cluster from a cache
	Update(*cache.Cache) error

	// The Get operations extract internal types from the Cluster
	// The returned Time value signifies the latest timestamp of all metrics on the cluster
	// TODO(alex): Returning pointers is NOT safe, will be addressed in a later PR

	GetAllClusterData() (*ClusterInfo, time.Time, error)
	// Parameter: Internal Hostname of the node
	GetAllNodeData(string) (*NodeInfo, time.Time, error)
	// Parameters: Namespace, Pod Name
	GetAllPodData(string, string) (*PodInfo, time.Time, error)
}

type realCluster struct {
	// Implementation of Cluster.
	// Timestamp signifies the latest timestamp of any metric
	// that is currently present in the realCluster
	timestamp time.Time
	ClusterInfo
}

// Internal Types
// REST consumption requires conversion to the corresponding versioned API types

type InfoType struct {
	Metrics map[string]*store.TimeStore // key: Metric Name
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
