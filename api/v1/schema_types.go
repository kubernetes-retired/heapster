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

package v1

import (
	"time"
)

type InfoType struct {
	// Generic object that contains metrics and labels
	// Embedded in relevant Info Objects
	Metrics map[string][]*MetricTimeseries `json:"metrics,omitempty"` // key: Metric name
	Labels  map[string]string              `json:"labels,omitempty"`  // key: Label
}

type ClusterInfo struct {
	// Cluster Information, links to Namespaces and Nodes
	InfoType
	Namespaces map[string]*NamespaceInfo `json:"namespaces,omitempty"` // key: Namespace Name
	Nodes      map[string]*NodeInfo      `json:"nodes,omitempty"`      // key: Hostname
}

type NamespaceInfo struct {
	// Namespace Information, links to Pods
	InfoType
	Pods map[string]*PodInfo `json:"pods,omitempty"` // key: Pod Name
}

type NodeInfo struct {
	// Node Information, links to Pods and Free Containers
	InfoType
	Pods           map[string]*PodInfo       `json:"pods,omitempty"`            // key: Pod Name
	FreeContainers map[string]*ContainerInfo `json:"free_containers,omitempty"` // key: Container Name
}

type PodInfo struct {
	// Pod Information, links to Containers
	InfoType
	UID        string                    `json:"uid,omitempty"`
	Containers map[string]*ContainerInfo `json:"containers,omitempty"` // key: Container Name
}

type ContainerInfo struct {
	// Container Information
	InfoType
}

type MetricTimeseries struct {
	Timestamp time.Time `json:"timestamp"`
	Value     uint64    `json:"value"`
}
