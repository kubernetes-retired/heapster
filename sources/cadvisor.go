package sources

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang/glog"
	cadvisorClient "github.com/google/cadvisor/client"
	cadvisor "github.com/google/cadvisor/info"
)

type cadvisorSource struct {
	containers []AnonContainer
	lastQuery  time.Time
}

func (self *cadvisorSource) addContainer(container *Container, hostname string) {
	self.containers = append(self.containers, AnonContainer{hostname, container})
}

func (self *cadvisorSource) processStat(hostname string, containerInfo *cadvisor.ContainerInfo) error {
	container := &Container{
		Name:  containerInfo.Name,
		Spec:  containerInfo.Spec,
		Stats: containerInfo.Stats,
	}
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}
	self.addContainer(container, hostname)
	return nil
}

func (self *cadvisorSource) getCadvisorData(hostname, ip, port, container string) error {
	client, err := cadvisorClient.NewClient("http://" + ip + ":" + port)
	if err != nil {
		return err
	}
	allContainers, err := client.SubcontainersInfo("/", &cadvisor.ContainerInfoRequest{int(time.Since(self.lastQuery) / time.Second)})
	if err != nil {
		glog.Errorf("failed to get stats from cadvisor on host %s with ip %s - %s\n", hostname, ip, err)
		return nil
	}

	for _, containerInfo := range allContainers {
		self.processStat(hostname, &containerInfo)
	}

	return nil
}

func (self *cadvisorSource) fetchData(cadvisorHosts *CadvisorHosts) ([]AnonContainer, error) {
	for hostname, ip := range cadvisorHosts.Hosts {
		err := self.getCadvisorData(hostname, ip, strconv.Itoa(cadvisorHosts.Port), "/")
		if err != nil {
			return nil, fmt.Errorf("Failed to get cAdvisor data from host %q: %v", hostname, err)
		}
	}
	return self.containers, nil
}

func newCadvisorSource() *cadvisorSource {
	return &cadvisorSource{
		containers: make([]AnonContainer, 0),
		lastQuery:  time.Now(),
	}
}
