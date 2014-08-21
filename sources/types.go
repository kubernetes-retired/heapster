package sources

import (
	"time"

	"github.com/google/cadvisor/info"
)

type ContainerHostnameMap map[string][]Container

type Source interface {
	FetchData() (ContainerHostnameMap, error)
}

type Container struct {
	Timestamp time.Time `json:"timestamp"`

	Name string `json:"name,omitempty"`

	Aliases []string `json:"aliases,omitempty"`

	// Historical statistics gathered from the container.
	Stats []*info.ContainerStats `json:"stats,omitempty"`

	// The isolation used in the container.
	Spec *info.ContainerSpec `json:"spec,omitempty"`
}
