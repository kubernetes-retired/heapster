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

package info

import (
	"time"
)

type InfoType struct {
	Metrics map[string][]*MetricTimeseries `json:"metrics,omitempty"`
	Labels  map[string]string              `json:"labels,omitempty"`
}

type ClusterInfo struct {
	InfoType
	Namespaces map[string]*NamespaceInfo `json:"namespaces,omitempty"`
	Nodes      map[string]*NodeInfo      `json:"nodes,omitempty"`
}

type NamespaceInfo struct {
	InfoType
	Pods map[string]*PodInfo `json:"pods,omitempty"`
}

type NodeInfo struct {
	InfoType
	Pods           map[string]*PodInfo       `json:"pods,omitempty"`
	FreeContainers map[string]*ContainerInfo `json:"free_containers"`
}

type PodInfo struct {
	InfoType
	UID        string                    `json:"uid,omitempty"`
	Containers map[string]*ContainerInfo `json:"containers,omitempty"`
}

type ContainerInfo struct {
	InfoType
}

type MetricTimeseries struct {
	Timestamp time.Time `json:"timestamp"`
	Value     uint64    `json:"value"`
}
