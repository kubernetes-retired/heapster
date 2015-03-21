// Copyright 2014 Google Inc. All Rights Reserved.
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

package api

import cadvisor "github.com/google/cadvisor/info/v1"

// PodState is the state of a pod, used as either input (desired state) or output (current state)
type Pod struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	// TODO(vishh): Rename to UID.
	ID             string            `json:"id,omitempty"`
	Hostname       string            `json:"hostname,omitempty"`
	Containers     []Container       `json:"containers"`
	Status         string            `json:"status,omitempty"`
	PodIP          string            `json:"pod_ip,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	HostPublicIP   string            `json:"host_public_ip,omitempty"`
	HostInternalIP string            `json:"host_internal_ip,omitempty"`
}

type AggregateData struct {
	Pods       []Pod
	Containers []Container
	Machine    []Container
}

type Container struct {
	Hostname string
	Name     string
	// TODO(vishh): Consider defining an internal Spec and Stats API to guard against
	// changed to cadvisor API.
	Spec  cadvisor.ContainerSpec
	Stats []*cadvisor.ContainerStats
}

func NewContainer() *Container {
	return &Container{Stats: make([]*cadvisor.ContainerStats, 0)}
}

// An external node represents a host which is running cadvisor. Heapster will expect
// data in this format via its external files API.
type ExternalNode struct {
	Name string `json:"name,omitempty"`
	IP   string `json:"ip,omitempty"`
}

// ExternalNodeList represents a list of all hosts that are running cadvisor which heapster will monitor.
type ExternalNodeList struct {
	Items []ExternalNode `json:"items,omitempty"`
}
