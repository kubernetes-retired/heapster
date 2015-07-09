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

package sinks

import (
	"container/list"
	"flag"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/api"
)

var argMaxStorageDuration = flag.Duration("sink_memory_ttl", 1*time.Hour, "Time duration for which stats should be cached if the memory sink is used")

type MemorySink struct {
	containersData     *list.List
	oldestData         time.Time
	maxStorageDuration time.Duration
}

type entry struct {
	timestamp time.Time
	data      interface{}
}

func (self *MemorySink) reapOldData() {
	if self.containersData.Len() == 0 || time.Since(self.oldestData) < self.maxStorageDuration {
		return
	}
	// TODO(vishh): Reap old data.
}

func (self *MemorySink) Store(data interface{}) error {
	if data, ok := data.([]api.Pod); ok {
		for _, value := range data {
			self.containersData.PushFront(entry{time.Now(), value})
			if self.containersData.Len() == 1 {
				self.oldestData = time.Now()
			}
		}
	}
	self.reapOldData()
	return nil
}

func (self *MemorySink) DebugInfo() string {
	desc := "Sink type: Memory\n"
	desc += fmt.Sprintf("\tMaximum storage duration: %v\n", self.maxStorageDuration)
	desc += "\n"
	return desc
}

func NewMemorySink() Sink {
	return &MemorySink{
		containersData:     list.New(),
		oldestData:         time.Now(),
		maxStorageDuration: *argMaxStorageDuration,
	}
}
