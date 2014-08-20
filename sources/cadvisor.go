package sources

import (
	"flag"
	"fmt"
	"net/http"
	"time"
)


var argCadvisorPort = flag.Int("cadvisor_port", 4194, "Port of cAdvisor")
var argMaster = flag.String("kubernetes_master", "", "Kubernetes master IP")
var argPollDuration = flag.Duration("cadvisor_poll_duration", 10*time.Second, "Port of cAdvisor")

type cadvisorSource struct {
	hosts []string
	pollDuration time.Duration
	master string
	cadvisorPort int
	recentStats []DataEntry
}


func (self *cadvisorSource) updateHosts() error {
	resp, err := http.Get(self.master+"/api/v1beta1/minions")
	if err != nil {
		return fmt.Errorf("Failed to get list of minions from Master %s : %v", self.master, err)
	}
	fmt.Printf("%+v", resp.Body)
	return nil
}

func (self *cadvisorSource) FetchData() error {
	if err := self.updateHosts(); err != nil {
		return err
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
	return &cadvisorSource{
		pollDuration: *argPollDuration,
		master: *argMaster,
		cadvisorPort: *argCadvisorPort,
	}, nil
}
