package sources

import (
	"flag"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/cadvisor/info"
)

var (
	argCadvisorPort = flag.Int("cadvisor_port", 4194, "Port of cAdvisor")
)

type CadvisorSource struct {
	cadvisorPort         string
	containerHostnameMap ContainerHostnameMap
	lastQuery            time.Time
}

func (self *CadvisorSource) addContainerToMap(container *Container, hostname string) {
	// TODO(vishh): Add a lock here to enable polling multiple hosts at the same time.
	self.containerHostnameMap[hostname] = append(self.containerHostnameMap[hostname], *container)
}

func (self *CadvisorSource) getCadvisorStatsUrl(host, container string) string {
	values := url.Values{}
	values.Add("num_stats", strconv.Itoa(int(time.Since(self.lastQuery)/time.Second)))
	values.Add("num_samples", strconv.Itoa(0))
	return "http://" + host + ":" + self.cadvisorPort + "/api/v1.0/containers" + container + "?" + values.Encode()
}

func (self *CadvisorSource) processStat(hostname string, containerInfo *info.ContainerInfo) error {
	container := &Container{
		Timestamp: time.Now(),
		Name:      containerInfo.Name,
		Aliases:   containerInfo.Aliases,
	}
	container.Stats = containerInfo.Stats
	container.Spec = containerInfo.Spec
	self.addContainerToMap(container, hostname)
	return nil
}

func (self *CadvisorSource) getCadvisorData(hostname, ip, container string) error {
	var containerInfo info.ContainerInfo
	req, err := http.NewRequest("GET", self.getCadvisorStatsUrl(ip, container), nil)
	if err != nil {
		return err
	}
	err = PostRequestAndGetValue(&http.Client{}, req, &containerInfo)
	if err != nil {
		return err
	}
	self.processStat(hostname, &containerInfo)
	for _, container := range containerInfo.Subcontainers {
		self.getCadvisorData(hostname, ip, container.Name)
	}
	return nil
}

func (self *CadvisorSource) FetchData(hosts map[string]string) (ContainerHostnameMap, error) {
	for hostname, ip := range hosts {
		err := self.getCadvisorData(hostname, ip, "/")
		if err != nil {
			return nil, err
		}
	}
	return self.containerHostnameMap, nil
}

func NewCadvisorSource() (*CadvisorSource, error) {
	return &CadvisorSource{
		cadvisorPort:         strconv.Itoa(*argCadvisorPort),
		containerHostnameMap: make(ContainerHostnameMap, 0),
		lastQuery:            time.Now(),
	}, nil
}
