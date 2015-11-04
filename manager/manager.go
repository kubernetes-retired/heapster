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
)

// Manager provides an interface to control the core of heapster.
// Implementations are not required to be thread safe.
type Manager interface {
	Start()
	Stop()
}

type realManager struct {
	sources    sources.SourceManager
	processors []core.DataProcessor
	sinks      sinks.SinkManager
	lastSync   time.Time
	resolution time.Duration
	stopChan   chan struct{}
}

func NewManager(sources sources.SourceManager, processors []core.DataProcessor, sinks sinks.SinkManager, res time.Duration) (Manager, error) {
	return &realManager{
		sources:    sources,
		processors: processors,
		sinks:      sinks,
		lastSync:   time.Now().Round(res),
		resolution: res,
		stopChan:   make(chan struct{}),
	}, nil
}

func (rm *realManager) Start() {
	go rm.Housekeep()
}

func (rm *realManager) Stop() {
	rm.stopChan <- struct{}{}
}

func (rm *realManager) Housekeep() {
	for {
		start := rm.lastSync
		end := start.Add(rm.resolution)
		timeToNextSync := end.Sub(time.Now())
		// TODO: consider adding some delay here
		select {
		case <-time.After(timeToNextSync):
			rm.housekeep(start, end)
			rm.lastSync = end
		case <-rm.stopChan:
			return
		}
	}
}

func (rm *realManager) housekeep(start, end time.Time) {
	data := rm.sources.ScrapeSources(start, end)
	for _, p := range rm.processors {
		data = p.Process(data)
	}
	rm.sinks.ExportData(data)
}
