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

package manager

import (
	"fmt"
	"sync"
	"time"

	sink_api "github.com/GoogleCloudPlatform/heapster/sinks"
	cache "github.com/GoogleCloudPlatform/heapster/sinks/cache"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/golang/glog"
)

// Manager provides an interface to control the core of heapster.
// Implementations are not required to be thread safe.
type Manager interface {
	// Housekeep collects data from all the configured sources and
	// stores the data to all the configured sinks.
	Housekeep()
}

type realManager struct {
	sources     []source_api.Source
	sinkManager sink_api.ExternalSinkManager
	cache       cache.Cache
	lastSync    time.Time
	resolution  time.Duration
}

type syncData struct {
	data  source_api.AggregateData
	mutex sync.Mutex
}

func NewManager(sources []source_api.Source, sinkManager sink_api.ExternalSinkManager, res, bufferDuration time.Duration) (Manager, error) {
	return &realManager{
		sources:     sources,
		sinkManager: sinkManager,
		cache:       cache.NewCache(bufferDuration),
		lastSync:    time.Now(),
		resolution:  res,
	}, nil
}

func (rm *realManager) scrapeSource(s source_api.Source, start, end time.Time, sd *syncData, errChan chan<- error) {
	glog.V(2).Infof("attempting to get data from source %q", s.Name())
	data, err := s.GetInfo(start, end, rm.resolution)
	if err != nil {
		err = fmt.Errorf("failed to get information from source %q - %v", s.Name(), err)
	} else {
		sd.mutex.Lock()
		defer sd.mutex.Unlock()
		sd.data.Merge(&data)
	}
	errChan <- err
}

func (rm *realManager) Housekeep() {
	errChan := make(chan error, len(rm.sources))
	var sd syncData
	start := rm.lastSync
	end := time.Now()
	rm.lastSync = start
	glog.V(2).Infof("starting to scrape data from sources")
	for idx := range rm.sources {
		s := rm.sources[idx]
		go rm.scrapeSource(s, start, end, &sd, errChan)
	}
	var errors []string
	for i := 0; i < len(rm.sources); i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err.Error())
		}
	}
	glog.V(2).Infof("completed scraping data from sources. Errors: %v", errors)
	if err := rm.cache.StorePods(sd.data.Pods); err != nil {
		errors = append(errors, err.Error())
	}
	if err := rm.cache.StoreContainers(sd.data.Machine); err != nil {
		errors = append(errors, err.Error())
	}
	if err := rm.cache.StoreContainers(sd.data.Containers); err != nil {
		errors = append(errors, err.Error())
	}
	if err := rm.sinkManager.Store(sd.data); err != nil {
		errors = append(errors, err.Error())
	}
	if len(errors) > 0 {
		glog.V(1).Infof("housekeeping resulted in following errors: %v", errors)
	}
}
