package main

import (
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/vishh/caggregator/sources"
)

var argPollDuration = flag.Duration("cadvisor_poll_duration", 10*time.Second, "Port of cAdvisor")

func main() {
	flag.Parse()
	var stop chan bool
	err := doWork(stop)
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func doWork(stop chan bool) error {
	kubeMasterSource, err := sources.NewKubeMasterSource()
	if err != nil {
		return err
	}
	cadvisorSource, err := sources.NewCadvisorSource()
	if err != nil {
		os.Exit(1)
	}
	var containersHistory []sources.ContainerHostnameMap
	ticker := time.NewTicker(*argPollDuration)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
			minions, err := kubeMasterSource.ListMinions()
			if err != nil {
				return err
			}
			data, err := cadvisorSource.FetchData(minions)
			if err != nil {
				return err
			}
			containersHistory = append(containersHistory, data)
		}
	}
	return nil
}
