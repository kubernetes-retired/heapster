package main

import (
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/vishh/caggregator/sinks"
	"github.com/vishh/caggregator/sources"
)

var argPollDuration = flag.Duration("poll_duration", 10*time.Second, "Polling duration")

func main() {
	flag.Parse()
	err := doWork()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func doWork() error {
	kubeMasterSource, err := sources.NewKubeMasterSource()
	if err != nil {
		return err
	}
	cadvisorSource, err := sources.NewCadvisorSource()
	if err != nil {
		return err
	}
	sink, err := sinks.NewSink()
	if err != nil {
		return err
	}
	ticker := time.NewTicker(*argPollDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			minions, err := kubeMasterSource.ListMinions()
			if err != nil {
				return err
			}
			data, err := cadvisorSource.FetchData(minions)
			if err != nil {
				return err
			}
			pods, err := kubeMasterSource.ListPods()
			if err != nil {
				return err
			}
			for idx, pod := range pods {
				for cIdx, container := range pod.Containers {
					containerInfoArray := data[pod.Hostname][container.ID]
					for _, containerInfo := range containerInfoArray {
						pods[idx].Containers[cIdx].Stats = append(pods[idx].Containers[cIdx].Stats, containerInfo.Stats...)
					}
				}
			}
			if err := sink.StoreData(pods); err != nil {
				return err
			}
		}
	}
	return nil
}
