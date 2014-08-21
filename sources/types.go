package sources

import (
	"time"

	"github.com/google/cadvisor/info"
)

type HostnameContainersMap map[string]map[string][]Container

type Container struct {
	Timestamp time.Time `json:"timestamp"`

	Aliases []string `json:"aliases,omitempty"`

	// Historical statistics gathered from the container.
	Stats []*info.ContainerStats `json:"stats,omitempty"`

	// The isolation used in the container.
	Spec *info.ContainerSpec `json:"spec,omitempty"`
}

type ContainerDesc struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// PodState is the state of a pod, used as either input (desired state) or output (current state)
type Pod struct {
	Containers []ContainerDesc   `json:"containers"`
	Status     string            `json:"status,omitempty"`
	PodIP      string            `json:"podIP,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}
