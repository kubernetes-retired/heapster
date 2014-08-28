package sources

import (
	"flag"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info"
)

var (
	argCadvisorPort = flag.Int("cadvisor_port", 4194, "Port of cAdvisor")
)

type CadvisorSource struct {
	cadvisorPort          string
	hostnameContainersMap HostnameContainersMap
	lastQuery             time.Time
}

func (self *CadvisorSource) addContainerToMap(container *Container, hostname string) {
	// TODO(vishh): Add a lock here to enable updating multiple hosts at the same time.
	if self.hostnameContainersMap[hostname] == nil {
		self.hostnameContainersMap[hostname] = make(IdToContainerMap, 0)
	}
	self.hostnameContainersMap[hostname][container.ID] = container
}

func (self *CadvisorSource) getCadvisorStatsUrl(host, container string) string {
	values := url.Values{}
	values.Add("num_stats", strconv.Itoa(int(time.Since(self.lastQuery)/time.Second)))
	values.Add("num_samples", strconv.Itoa(0))
	return "http://" + host + ":" + self.cadvisorPort + "/api/v1.0/containers" + container + "?" + values.Encode()
}

func (self *CadvisorSource) processStat(hostname string, containerInfo *cadvisor.ContainerInfo) error {
	container := &Container{
		Name: containerInfo.Name,
		ID:   filepath.Base(containerInfo.Name),
	}
	container.Stats = containerInfo.Stats
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}
	self.addContainerToMap(container, hostname)
	return nil
}

func (self *CadvisorSource) getCadvisorData(hostname, ip, container string) error {
	var containerInfo cadvisor.ContainerInfo
	req, err := http.NewRequest("GET", self.getCadvisorStatsUrl(ip, container), nil)
	if err != nil {
		return err
	}
	err = PostRequestAndGetValue(&http.Client{}, req, &containerInfo)
	if err != nil {
		glog.Errorf("failed to get stats from cadvisor on host %s with ip %s - %s\n", hostname, ip, err)
		return nil
	}
	self.processStat(hostname, &containerInfo)
	for _, container := range containerInfo.Subcontainers {
		self.getCadvisorData(hostname, ip, container.Name)
	}
	return nil
}

func (self *CadvisorSource) FetchData(hosts map[string]string) (HostnameContainersMap, error) {
	for hostname, ip := range hosts {
		err := self.getCadvisorData(hostname, ip, "/")
		if err != nil {
			return nil, err
		}
	}
	return self.hostnameContainersMap, nil
}

func NewCadvisorSource() (*CadvisorSource, error) {
	return &CadvisorSource{
		cadvisorPort:          strconv.Itoa(*argCadvisorPort),
		hostnameContainersMap: make(HostnameContainersMap, 0),
		lastQuery:             time.Now(),
	}, nil
}
