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
	lastQuery time.Time
}

func (self *cadvisorSource) processStat(hostname string, containerInfo *cadvisor.ContainerInfo) RawContainer {
	container := Container{
		Name:  containerInfo.Name,
		Spec:  containerInfo.Spec,
		Stats: containerInfo.Stats,
	}
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}

	return RawContainer{hostname, container}
}

func (self *cadvisorSource) getAllCadvisorData(hostname, ip, port, container string) (containers []RawContainer, nodeInfo RawContainer, err error) {
	client, err := cadvisorClient.NewClient("http://" + ip + ":" + port)
	if err != nil {
		return
	}
	allContainers, err := client.SubcontainersInfo("/", 
		&cadvisor.ContainerInfoRequest{NumStats: int(time.Since(self.lastQuery) / time.Second)})
	if err != nil {
		glog.Errorf("failed to get stats from cadvisor on host %s with ip %s - %s\n", hostname, ip, err)
		return
	}

	for _, containerInfo := range allContainers {
		rawContainer := self.processStat(hostname, &containerInfo)
		if containerInfo.Name == "/" {
			nodeInfo = rawContainer
		} else {
			containers = append(containers, rawContainer)
		}
	}

	return
}

func (self *cadvisorSource) fetchData(cadvisorHosts *CadvisorHosts) (rawContainers []RawContainer, nodesInfo []RawContainer, err error) {
	for hostname, ip := range cadvisorHosts.Hosts {
		containers, nodeInfo, err := self.getAllCadvisorData(hostname, ip, strconv.Itoa(cadvisorHosts.Port), "/")
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to get cAdvisor data from host %q: %v", hostname, err)
		}
		rawContainers = append(rawContainers, containers...)
		nodesInfo = append(nodesInfo, nodeInfo)
	}

	return
}

func newCadvisorSource() *cadvisorSource {
	return &cadvisorSource{
		lastQuery: time.Now(),
	}
}
