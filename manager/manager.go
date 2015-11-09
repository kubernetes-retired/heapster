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
	"time"

	"k8s.io/heapster/core"
	"k8s.io/heapster/sinks"
	"k8s.io/heapster/sources"

	"github.com/golang/glog"
)

const (
	scrapeOffset          = 5 * time.Second
	maxParallelHousekeeps = 3
)

// Manager provides an interface to control the core of heapster.
// Implementations are not required to be thread safe.
type Manager interface {
	Start()
	Stop()
}

type realManager struct {
	source                 sources.MetricsSource
	processors             []core.DataProcessor
	sink                   sinks.DataSink
	resolution             time.Duration
	stopChan               chan struct{}
	housekeepSemaphoreChan chan struct{}
	housekeepTimeout       time.Duration
}

func NewManager(source sources.MetricsSource, processors []core.DataProcessor, sink sinks.DataSink, res time.Duration) (Manager, error) {
	manager := realManager{
		source:                 source,
		processors:             processors,
		sink:                   sink,
		resolution:             res,
		stopChan:               make(chan struct{}),
		housekeepSemaphoreChan: make(chan struct{}, maxParallelHousekeeps),
		housekeepTimeout:       res / 2,
	}

	for i := 0; i < maxParallelHousekeeps; i++ {
		manager.housekeepSemaphoreChan <- struct{}{}
	}

	return &manager, nil
}

func (rm *realManager) Start() {
	go rm.Housekeep()
}

func (rm *realManager) Stop() {
	rm.stopChan <- struct{}{}
}

func (rm *realManager) Housekeep() {
	for {
		// Always try to get the newest metrics
		now := time.Now()
		start := now.Truncate(rm.resolution)
		end := start.Add(rm.resolution)
		timeToNextSync := end.Add(scrapeOffset).Sub(now)

		select {
		case <-time.After(timeToNextSync):
			rm.housekeep(start, end)
		case <-rm.stopChan:
			rm.sink.Stop()
			return
		}
	}
}

func (rm *realManager) housekeep(start, end time.Time) {
	if !start.Before(end) {
		glog.Warningf("Wrong time provided to housekeep start:%s end: %s", start, end)
		return
	}

	select {
	case <-rm.housekeepSemaphoreChan:
		// ok, good to go

	case <-time.After(rm.housekeepTimeout):
		glog.Warningf("Spent too long waiting for housekeeping to start")
		return
	}

	go func(rm *realManager) {
		// should always give back the semaphore
		defer func() { rm.housekeepSemaphoreChan <- struct{}{} }()

		data := rm.source.ScrapeMetrics(start, end)
		for _, p := range rm.processors {
			data = p.Process(data)
		}
		rm.sink.ExportData(data)
	}(rm)
}
