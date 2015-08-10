// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datasource

import (
	"time"

	"k8s.io/heapster/sources/api"

	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info/v1"
)

func parseStat(containerInfo *cadvisor.ContainerInfo, start time.Time, resolution time.Duration, align bool) *api.Container {
	if len(containerInfo.Stats) == 0 {
		return nil
	}
	container := &api.Container{
		Name:  containerInfo.Name,
		Spec:  containerInfo.Spec,
		Stats: sampleContainerStats(containerInfo.Stats, start, resolution, align),
	}
	if len(containerInfo.Aliases) > 0 {
		container.Name = containerInfo.Aliases[0]
	}

	return container
}

func sampleContainerStats(stats []*cadvisor.ContainerStats, start time.Time, resolution time.Duration, align bool) []*cadvisor.ContainerStats {
	if len(stats) == 0 {
		return stats
	}
	filteredStats := []*cadvisor.ContainerStats{}
	var nextTimestamp time.Time
	if align {
		// Next timestamp which is multiple of resolution.
		nextTimestamp = start.Add(-time.Nanosecond).Truncate(resolution).Add(resolution)
	} else {
		for _, s := range stats {
			if s != nil {
				nextTimestamp = s.Timestamp
				break
			}
		}
	}
	for _, s := range stats {
		if s == nil {
			continue
		}
		if s.Timestamp.Before(nextTimestamp) {
			continue
		}
		if align {
			s.Timestamp = nextTimestamp
		}
		filteredStats = append(filteredStats, s)
		nextTimestamp = nextTimestamp.Add(resolution)
	}
	glog.V(5).Infof("got %d stats, returning %d after filtering", len(stats), len(filteredStats))
	return filteredStats
}
