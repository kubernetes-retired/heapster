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

	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info/v1"
)

func sampleContainerStats(stats []*cadvisor.ContainerStats, resolution time.Duration) []*cadvisor.ContainerStats {
	if len(stats) == 0 {
		return stats
	}
	filteredStats := []*cadvisor.ContainerStats{}
	var nextTimestamp time.Time
	for index := range stats {
		if stats[index] == nil {
			continue
		}
		if len(filteredStats) == 0 {
			nextTimestamp = stats[index].Timestamp
		}
		if stats[index].Timestamp.Before(nextTimestamp) {
			continue
		}
		filteredStats = append(filteredStats, stats[index])
		nextTimestamp = nextTimestamp.Add(resolution)
	}
	glog.V(5).Infof("got %d stats, returning %d after filtering", len(stats), len(filteredStats))
	return filteredStats
}
