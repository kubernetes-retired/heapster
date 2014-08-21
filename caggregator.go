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
	glog.Info("cAggregator is running")
	flag.Parse()
	cadvisorSource, err := sources.NewCadvisorSource()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	var containersHistory []sources.ContainerHostnameMap
	ticker := time.NewTicker(*argPollDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			data, err := cadvisorSource.FetchData()
			if err != nil {
				glog.Error(err)
				os.Exit(1)
			}
			containersHistory = append(containersHistory, data)
		}
	}
	os.Exit(0)
}
