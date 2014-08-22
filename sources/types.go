package sources

import cadvisor "github.com/google/cadvisor/info"

type IdToContainerInfoMap map[string][]ContainerInfo
type HostnameContainersMap map[string]IdToContainerInfoMap

type ContainerInfo struct {
	// Historical statistics gathered from the container.
	Stats []*cadvisor.ContainerStats `json:"stats,omitempty"`
}

type Container struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	ContainerInfo
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
