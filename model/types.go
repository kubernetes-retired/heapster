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
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	"github.com/GoogleCloudPlatform/heapster/store"
)

type Cluster interface {
	// The Update operation populates the Cluster from a cache.
	Update(cache.Cache) error

	// The Get operations extract internal types from the Cluster.
	// The returned time.Time values signify the latest metric timestamp in the cluster.
	// TODO(alex): Get Operations are being reconstructed for [Schema 7]
}

// realCluster is an implementation of the Cluster interface.
// timestamp marks the latest timestamp of any metric present in the realCluster.
// tsConstructor generates a new empty TimeStore, used for storing historical data.
type realCluster struct {
	timestamp     time.Time
	lock          sync.RWMutex
	tsConstructor func() store.TimeStore
	resolution    time.Duration
	ClusterInfo
}

// Internal Types
// REST consumption requires conversion to the corresponding external API types.

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

// Supported metric names, used as keys for all map[string]*store.TimeStore
const cpuLimit = "cpu/limit"
const cpuUsage = "cpu/usage"
const memLimit = "memory/limit"
const memUsage = "memory/usage"
const memWorking = "memory/working"
const fsLimit = "fs/limit"
const fsUsage = "fs/usage"
