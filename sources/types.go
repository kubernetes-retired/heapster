package sources

import (
	cadvisor "github.com/google/cadvisor/info"
)

type IdToContainerMap map[string]*Container
type HostnameContainersMap map[string]IdToContainerMap

type Container struct {
	ID    string                     `json:"id,omitempty"`
	Name  string                     `json:"name,omitempty"`
	Stats []*cadvisor.ContainerStats `json:"stats,omitempty"`
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

type AnonContainer struct {
	Hostname string `json:"hostname,omitempty"`
	*Container
}
