package sources

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

var (
	argHostsFile = flag.String("hosts_file", "/var/run/heapster/hosts", "")
)

type ExternalSource struct {
	hostsFile string
	cadvisor  *cadvisorSource
}

func (self *ExternalSource) getCadvisorHosts() (*CadvisorHosts, error) {
	contents, err := ioutil.ReadFile(self.hostsFile)
	if err != nil {
		return nil, err
	}
	var cadvisorHosts CadvisorHosts
	err = json.Unmarshal(contents, &cadvisorHosts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal contents of file %s. Error: %s", self.hostsFile, err)
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
	if len(*argHostsFile) == 0 {
		return nil, fmt.Errorf("hosts_file flag invalid")
	}
	if _, err := os.Stat(*argHostsFile); err != nil {
		return nil, fmt.Errorf("Cannot stat hosts_file %s. Error: %s", *argHostsFile, err)
	}
	cadvisorSource := newCadvisorSource()
	return &ExternalSource{
		hostsFile: *argHostsFile,
		cadvisor:  cadvisorSource,
	}, nil
}
