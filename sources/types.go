package sources

import (
	"flag"
	cadvisor "github.com/google/cadvisor/info"
)

var (
	argMaster     = flag.String("kubernetes_master", "", "Kubernetes master IP")
	argMasterAuth = flag.String("kubernetes_master_auth", "", "username:password to access the master")
)

type IdToContainerMap map[string]*Container
type HostnameContainersMap map[string]IdToContainerMap

type Container struct {
	ID    string                     `json:"id,omitempty"`
	Name  string                     `json:"name,omitempty"`
	Stats []*cadvisor.ContainerStats `json:"stats,omitempty"`
}

func newContainer() *Container {
	return &Container{Stats: make([]*cadvisor.ContainerStats, 0)}
}

// PodState is the state of a pod, used as either input (desired state) or output (current state)
type Pod struct {
	Name       string            `json:"name,omitempty"`
	Hostname   string            `json:"hostname,omitempty"`
	Containers []*Container      `json:"containers"`
	Status     string            `json:"status,omitempty"`
	PodIP      string            `json:"podIP,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type AnonContainer struct {
	Hostname string `json:"hostname,omitempty"`
	*Container
}

type CadvisorHosts struct {
	Port  int               `json:"port"`
	Hosts map[string]string `json:"hosts"`
}

type Source interface {
	GetPods() ([]Pod, error)
	GetContainerStats() (HostnameContainersMap, error)
}

func NewSource() (Source, error) {
	if len(*argMaster) > 0 {
		return newKubeSource()
	} else {
		return newExternalSource()
	}
}
