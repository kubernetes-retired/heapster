package sources

import (
	"time"

	cadvisor "github.com/google/cadvisor/info"
)

type HostnameContainersMap map[string]map[string][]ContainerInfo

type ContainerInfo struct {
	Aliases []string `json:"aliases,omitempty"`

	// Historical statistics gathered from the container.
	Stats []*cadvisor.ContainerStats `json:"stats,omitempty"`

	// The isolation used in the container.
	Spec *cadvisor.ContainerSpec `json:"spec,omitempty"`
}

type Container struct {
	ID   string          `json:"id,omitempty"`
	Name string          `json:"name,omitempty"`
	Info []ContainerInfo `json:"info,omitempty"`
}

// PodState is the state of a pod, used as either input (desired state) or output (current state)
type Pod struct {
	Name       string            `json:"name,omitempty"`
	Hostname   string            `json:"hostname,omitempty"`
	Containers []Container       `json:"containers"`
	Status     string            `json:"status,omitempty"`
	PodIP      string            `json:"podIP,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}
