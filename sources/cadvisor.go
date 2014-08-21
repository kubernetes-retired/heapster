package sources

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/google/cadvisor/info"
)

var (
	argCadvisorPort = flag.Int("cadvisor_port", 4194, "Port of cAdvisor")
	argMaster       = flag.String("kubernetes_master", "", "Kubernetes master IP")
	argMasterAuth   = flag.String("kubernetes_master_auth", "", "username:password to access the master")
)

type cadvisorSource struct {
	hosts                map[string]string
	master               string
	cadvisorPort         string
	containerHostnameMap ContainerHostnameMap
	authMasterUser       string
	authMasterPass       string
	lastQuery            time.Time
}

func (self *cadvisorSource) addContainerToMap(container *Container, hostname string) {
	// TODO(vishh): Add a lock here to enable polling multiple hosts at the same time.
	self.containerHostnameMap[hostname] = append(self.containerHostnameMap[hostname], *container)
}

func (self *cadvisorSource) postRequestAndGetValue(client *http.Client, req *http.Request, value interface{}) error {
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, value)
	if err != nil {
		return fmt.Errorf("Got '%s': %v", string(body), err)
	}
	return nil
}

func (self *cadvisorSource) masterListMinionsUrl() string {
	return self.master + "/api/v1beta1/minions"
}

func (self *cadvisorSource) updateHosts() error {
	var minions kube_api.MinionList
	req, err := http.NewRequest("GET", self.masterListMinionsUrl(), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(self.authMasterUser, self.authMasterPass)
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	err = self.postRequestAndGetValue(httpClient, req, &minions)
	if err != nil {
		return err
	}
	for _, value := range minions.Items {
		addrs, err := net.LookupIP(value.ID)
		if err == nil {
			self.hosts[value.ID] = addrs[0].String()
		}
	}
	return nil
}

func (self *cadvisorSource) getCadvisorStatsUrl(host, container string) string {
	values := url.Values{}
	values.Add("num_stats", strconv.Itoa(int(time.Since(self.lastQuery)/time.Second)))
	values.Add("num_samples", strconv.Itoa(0))
	return "http://" + host + ":" + self.cadvisorPort + "/api/v1.0/containers" + container + "?" + values.Encode()
}

func (self *cadvisorSource) processStat(hostname string, containerInfo *info.ContainerInfo) error {
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

func (self *cadvisorSource) getCadvisorData(hostname, ip, container string) error {
	var containerInfo info.ContainerInfo
	req, err := http.NewRequest("GET", self.getCadvisorStatsUrl(ip, container), nil)
	if err != nil {
		return err
	}
	err = self.postRequestAndGetValue(&http.Client{}, req, &containerInfo)
	if err != nil {
		return err
	}
	self.processStat(hostname, &containerInfo)
	for _, container := range containerInfo.Subcontainers {
		self.getCadvisorData(hostname, ip, container.Name)
	}
	return nil
}

func (self *cadvisorSource) FetchData() (ContainerHostnameMap, error) {
	if err := self.updateHosts(); err != nil {
		return nil, err
	}
	for hostname, ip := range self.hosts {
		err := self.getCadvisorData(hostname, ip, "/")
		if err != nil {
			return nil, err
		}
	}
	return self.containerHostnameMap, nil
}

func NewCadvisorSource() (Source, error) {
	if len(*argMaster) == 0 {
		return nil, fmt.Errorf("kubernetes_master flag not specified")
	}
	if len(*argMasterAuth) == 0 || len(strings.Split(*argMasterAuth, ":")) != 2 {
		return nil, fmt.Errorf("kubernetes_master_auth invalid")
	}
	authInfo := strings.Split(*argMasterAuth, ":")
	return &cadvisorSource{
		master:               "https://" + *argMaster,
		cadvisorPort:         strconv.Itoa(*argCadvisorPort),
		authMasterUser:       authInfo[0],
		authMasterPass:       authInfo[1],
		containerHostnameMap: make(ContainerHostnameMap, 0),
		hosts:                make(map[string]string, 0),
		lastQuery:            time.Now(),
	}, nil
}
