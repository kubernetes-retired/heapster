// Copyright 2015 Google Inc. All Rights Reserved.
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

package schema

import (
	"github.com/GoogleCloudPlatform/heapster/api/schema/info"
	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	"sync"
	"time"
)

type Cluster interface {
	Update(*cache.Cache) error

	GetAllClusterData() (*info.ClusterInfo, time.Time, error)
	/*
		GetNewClusterData(time.Time) (*ClusterInfo, time.Time, error)

		GetAllNodeData(string) (*NodeInfo, time.Time, error)
		GetNewNodeData(string, time.Time) (*NodeInfo, time.Time, error)

		GetAllPodData(string) (*PodInfo, time.Time, error)
		GetNewPodData(string) (*PodInfo, time.Time, error)
	*/
}

type realCluster struct {
	timestamp time.Time // Cluster timestamp signifies the last update from cache.
	lock      *sync.RWMutex
	info.ClusterInfo
}
