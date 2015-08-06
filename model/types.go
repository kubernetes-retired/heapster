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

	// The simple Get operations extract structural information from the Cluster.
	GetAvailableMetrics() []string
	GetNodes() []string
	GetNamespaces() []string
	GetPods(string) []string
	GetPodContainers(string, string) []string
	GetFreeContainers(string) []string

	// The GetXMetric operations extract timeseries from the Cluster.
	// The returned time.Time values signify the latest metric timestamp in the cluster.
	GetClusterMetric(ClusterMetricRequest) ([]store.TimePoint, time.Time, error)
	GetNodeMetric(NodeMetricRequest) ([]store.TimePoint, time.Time, error)
	GetNamespaceMetric(NamespaceMetricRequest) ([]store.TimePoint, time.Time, error)
	GetPodMetric(PodMetricRequest) ([]store.TimePoint, time.Time, error)
	GetPodContainerMetric(PodContainerMetricRequest) ([]store.TimePoint, time.Time, error)
	GetFreeContainerMetric(FreeContainerMetricRequest) ([]store.TimePoint, time.Time, error)

	// The GetXStats operations extract all derived stats for a single entity of the cluster.
	GetClusterStats() (map[string]StatBundle, time.Duration, error)
	GetNodeStats(NodeRequest) (map[string]StatBundle, time.Duration, error)
	GetNamespaceStats(NamespaceRequest) (map[string]StatBundle, time.Duration, error)
	GetPodStats(PodRequest) (map[string]StatBundle, time.Duration, error)
	GetPodContainerStats(PodContainerRequest) (map[string]StatBundle, time.Duration, error)
	GetFreeContainerStats(FreeContainerRequest) (map[string]StatBundle, time.Duration, error)
}

// realCluster is an implementation of the Cluster interface.
// timestamp marks the latest timestamp of any metric present in the realCluster.
// tsConstructor generates a new empty TimeStore, used for storing historical data.
type realCluster struct {
	timestamp      time.Time
	lock           sync.RWMutex
	dayConstructor func() store.DayStore
	tsConstructor  func() store.TimeStore
	resolution     time.Duration
	ClusterInfo
}

// Supported metric names, used as keys for all map[string]*store.TimeStore
const cpuLimit = "cpu-limit"
const cpuUsage = "cpu-usage"
const memLimit = "memory-limit"
const memUsage = "memory-usage"
const memWorking = "memory-working"
const fsLimit = "fs-limit"
const fsUsage = "fs-usage"

// Simple Request Types.
type MetricRequest struct {
	MetricName string
	Start      time.Time
	End        time.Time
}

type NodeRequest struct {
	NodeName string
}

type NamespaceRequest struct {
	NamespaceName string
}

type PodRequest struct {
	NamespaceName string
	PodName       string
}

type PodContainerRequest struct {
	NamespaceName string
	PodName       string
	ContainerName string
}

type FreeContainerRequest struct {
	NodeName      string
	ContainerName string
}

// Metric Request Types
type ClusterMetricRequest struct {
	MetricRequest
}

type NodeMetricRequest struct {
	NodeName string
	MetricRequest
}

type NamespaceMetricRequest struct {
	NamespaceName string
	MetricRequest
}

type PodMetricRequest struct {
	NamespaceName string
	PodName       string
	MetricRequest
}

type PodContainerMetricRequest struct {
	NamespaceName string
	PodName       string
	ContainerName string
	MetricRequest
}

type FreeContainerMetricRequest struct {
	NodeName      string
	ContainerName string
	MetricRequest
}

// Derived Stats Types

type Stats struct {
	Average    uint64
	Percentile uint64
	Max        uint64
}

type StatBundle struct {
	Minute Stats
	Hour   Stats
	Day    Stats
}

// Internal Types
type InfoType struct {
	Uptime  time.Duration
	Metrics map[string]*store.DayStore // key: Metric Name
	Labels  map[string]string          // key: Label
	// Context retains instantaneous state for a specific InfoType.
	// Currently used for calculating instantaneous metrics from cumulative counterparts.
	Context map[string]*store.TimePoint // key: metric name
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
