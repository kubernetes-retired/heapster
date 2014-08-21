package sources

import (
	"time"

	"github.com/google/cadvisor/info"
)

type HostnameContainersMap map[string]map[string][]ContainerInfo

type ContainerInfo struct {
	Timestamp time.Time `json:"timestamp"`

	Aliases []string `json:"aliases,omitempty"`

	// Historical statistics gathered from the container.
	Stats []*info.ContainerStats `json:"stats,omitempty"`

	// The isolation used in the container.
	Spec *info.ContainerSpec `json:"spec,omitempty"`
}

type Container struct {
	ID   string          `json:"id,omitempty"`
	Name string          `json:"name,omitempty"`
	Info []ContainerInfo `json:"info,omitempty"`
}

// PodState is the state of a pod, used as either input (desired state) or output (current state)
type Pod struct {
	Hostname   string            `json:"hostname,omitempty"`
	Containers []Container       `json:"containers"`
	Status     string            `json:"status,omitempty"`
	PodIP      string            `json:"podIP,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}
