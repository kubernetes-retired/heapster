package main

import (
	"flag"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sinks"
	"github.com/GoogleCloudPlatform/heapster/sources"
	"github.com/golang/glog"
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
	source, err := sources.NewSource()
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
			cadvisorData, err := source.GetContainerStats()
			if err != nil {
				return err
			}
			pods, err := source.GetPods()
			if err != nil {
				return err
			}
			for idx, pod := range pods {
				for cIdx, container := range pod.Containers {
					containerOnHost := cadvisorData[pod.Hostname]
					if containerOnHost != nil {
						if _, ok := containerOnHost[container.ID]; ok {
							pods[idx].Containers[cIdx].Stats = append(pods[idx].Containers[cIdx].Stats, containerOnHost[container.ID].Stats...)
							delete(containerOnHost, container.ID)
						}
					}
				}
			}
			if err := sink.StoreData(pods); err != nil {
				return err
			}
			// Store all the anonymous containers.
			for hostname, idToContainerMap := range cadvisorData {
				for _, container := range idToContainerMap {
					if container == nil {
						continue
					}
					anonContainer := sources.AnonContainer{
						Hostname:  hostname,
						Container: container,
					}
					if err := sink.StoreData(anonContainer); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
