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

package sources

import (
	"time"

	cadvisor "github.com/google/cadvisor/info"
)

type Container struct {
	Name  string                     `json:"name,omitempty"`
	Spec  cadvisor.ContainerSpec     `json:"spec,omitempty"`
	Stats []*cadvisor.ContainerStats `json:"stats,omitempty"`
}

func newContainer() *Container {
	return &Container{Stats: make([]*cadvisor.ContainerStats, 0)}
}

// PodState is the state of a pod, used as either input (desired state) or output (current state)
type Pod struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	// TODO(vishh): Rename to UID.
	ID         string            `json:"id,omitempty"`
	Hostname   string            `json:"hostname,omitempty"`
	Containers []*Container      `json:"containers"`
	Status     string            `json:"status,omitempty"`
	PodIP      string            `json:"podIP,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	HostIP     string            `json:"hostIP,omitempty"`
}

type RawContainer struct {
	Hostname string `json:"hostname,omitempty"`
	Container
}

// TODO(vishh): Rename this to something more generic
type ContainerData struct {
	Pods       []Pod
	Containers []RawContainer
	Machine    []RawContainer
}

type Source interface {
	// Fetches containers or pod information from all the nodes in the cluster.
	// Returns:
	// 1. podsOrContainers: A slice of Pod or a slice of RawContainer
	// 2. nodes: A slice of RawContainer, one for each node in the cluster, that contains
	// root cgroup information.
	GetInfo() (ContainerData, error)
	// Returns debug information for the source.
	DebugInfo() string
}

func NewSource(pollDuration time.Duration) (Source, error) {
	if len(*argMaster) > 0 {
		return newKubeSource(pollDuration)
	} else {
		return newExternalSource(pollDuration)
	}
}
