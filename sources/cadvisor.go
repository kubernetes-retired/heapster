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
	argPollDuration = flag.Duration("cadvisor_poll_duration", 10*time.Second, "Port of cAdvisor")
	argMasterAuth   = flag.String("kubernetes_master_auth", "", "username:password to access the master")
)

type cadvisorSource struct {
	hosts          map[string]string
	pollDuration   time.Duration
	master         string
	cadvisorPort   string
	recentStats    []DataEntry
	authMasterUser string
	authMasterPass string
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
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &minions)
	if err != nil {
		return fmt.Errorf("Got '%s': %v", string(body), err)
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
	values.Add("num_stats", string(self.pollDuration/time.Second))
	values.Add("num_samples", string(0))
	return "http://" + host + ":" + self.cadvisorPort + "/api/v1.0/containers" + container + "?" + values.Encode()
}

func (self *cadvisorSource) getCadvisorData(host, container string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", self.getCadvisorStatsUrl(host, container), nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	var containerInfo info.ContainerInfo
	err = json.Unmarshal(body, &containerInfo)
	if err != nil {
		return fmt.Errorf("Got '%s': %v", string(body), err)
	}
	// TODO(vishh): Process all the stats and store it in self.recentStats
	// TODO(vishh): Invoke getCadvisorStatsUrl recursively on all the subcontainers
	return nil
}

func (self *cadvisorSource) FetchData() error {
	if err := self.updateHosts(); err != nil {
		return err
	}
	for _, ip := range self.hosts {
		err := self.getCadvisorData(ip, "/")
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *cadvisorSource) GetAndFlushData() ([]DataEntry, error) {
	return nil, nil
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
		pollDuration:   *argPollDuration,
		master:         "https://" + *argMaster,
		cadvisorPort:   strconv.Itoa(*argCadvisorPort),
		authMasterUser: authInfo[0],
		authMasterPass: authInfo[1],
		hosts:          make(map[string]string, 0),
	}, nil
}
