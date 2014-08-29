package sources

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// While updating this, also update heapster/deploy/Dockerfile.
const hostsFile = "/var/run/heapster/hosts"

type ExternalSource struct {
	cadvisor  *cadvisorSource
}

func (self *ExternalSource) getCadvisorHosts() (*CadvisorHosts, error) {
	contents, err := ioutil.ReadFile(hostsFile)
	if err != nil {
		return nil, err
	}
	var cadvisorHosts CadvisorHosts
	err = json.Unmarshal(contents, &cadvisorHosts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal contents of file %s. Error: %s", hostsFile, err)
	}
	return &cadvisorHosts, nil
}

func (self *ExternalSource) GetPods() ([]Pod, error) {
	return []Pod{}, nil
}

func (self *ExternalSource) GetContainerStats() (HostnameContainersMap, error) {
	hosts, err := self.getCadvisorHosts()
	if err != nil {
		return nil, err
	}
	return self.cadvisor.fetchData(hosts)
}

func newExternalSource() (Source, error) {
	if _, err := os.Stat(hostsFile); err != nil {
		return nil, fmt.Errorf("Cannot stat hosts_file %s. Error: %s", hostsFile, err)
	}
	cadvisorSource := newCadvisorSource()
	return &ExternalSource{
		cadvisor:  cadvisorSource,
	}, nil
}
