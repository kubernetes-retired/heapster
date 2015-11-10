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

package sinks

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/heapster/core"
)

type dummySink struct {
	name        string
	mutex       sync.Mutex
	exportCount int
	stopped     bool
	latency     time.Duration
}

func (this *dummySink) Name() string {
	return this.name
}
func (this *dummySink) ExportData(*core.DataBatch) {
	this.mutex.Lock()
	this.exportCount++
	this.mutex.Unlock()

	time.Sleep(this.latency)
}

func (this *dummySink) Stop() {
	this.mutex.Lock()
	this.stopped = true
	this.mutex.Unlock()

	time.Sleep(this.latency)
}

func newDummySink(name string, latency time.Duration) *dummySink {
	return &dummySink{
		name:        name,
		latency:     latency,
		exportCount: 0,
		stopped:     false,
	}
}

func TestAllExportsInTime(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := newDummySink("s1", time.Second)
	sink2 := newDummySink("s2", time.Second)
	manager, _ := NewDataSinkManager([]DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	batch := core.DataBatch{
		Timestamp:  now,
		MetricSets: map[string]*core.MetricSet{},
	}

	manager.ExportData(&batch)
	manager.ExportData(&batch)
	manager.ExportData(&batch)

	elapsed := time.Now().Sub(now)
	if elapsed > 2*timeout+2*time.Second {
		t.Fatalf("3xExportData took too long: %s", elapsed)
	}
	sink1.mutex.Lock()
	defer sink1.mutex.Unlock()
	assert.Equal(t, 3, sink1.exportCount)

	sink2.mutex.Lock()
	defer sink2.mutex.Unlock()
	assert.Equal(t, 3, sink2.exportCount)
}

func TestOneExportInTime(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := newDummySink("s1", time.Second)
	sink2 := newDummySink("s2", 30*time.Second)
	manager, _ := NewDataSinkManager([]DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	batch := core.DataBatch{
		Timestamp:  now,
		MetricSets: map[string]*core.MetricSet{},
	}

	manager.ExportData(&batch)
	manager.ExportData(&batch)
	manager.ExportData(&batch)

	elapsed := time.Now().Sub(now)
	if elapsed > 2*timeout+2*time.Second {
		t.Fatalf("3xExportData took too long: %s", elapsed)
	}
	if elapsed < 2*timeout-1*time.Second {
		t.Fatalf("3xExportData took too short: %s", elapsed)
	}

	sink1.mutex.Lock()
	defer sink1.mutex.Unlock()
	assert.Equal(t, 3, sink1.exportCount)

	sink2.mutex.Lock()
	defer sink2.mutex.Unlock()
	assert.Equal(t, 1, sink2.exportCount)
}

func TestNoExportInTime(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := newDummySink("s1", 30*time.Second)
	sink2 := newDummySink("s2", 30*time.Second)
	manager, _ := NewDataSinkManager([]DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	batch := core.DataBatch{
		Timestamp:  now,
		MetricSets: map[string]*core.MetricSet{},
	}

	manager.ExportData(&batch)
	manager.ExportData(&batch)
	manager.ExportData(&batch)

	elapsed := time.Now().Sub(now)
	if elapsed > 2*timeout+2*time.Second {
		t.Fatalf("3xExportData took too long: %s", elapsed)
	}
	if elapsed < 2*timeout-1*time.Second {
		t.Fatalf("3xExportData took too short: %s", elapsed)
	}

	sink1.mutex.Lock()
	defer sink1.mutex.Unlock()
	assert.Equal(t, 1, sink1.exportCount)

	sink2.mutex.Lock()
	defer sink2.mutex.Unlock()
	assert.Equal(t, 1, sink2.exportCount)
}

func TestStop(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := newDummySink("s1", 30*time.Second)
	sink2 := newDummySink("s2", 30*time.Second)
	manager, _ := NewDataSinkManager([]DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	manager.Stop()
	elapsed := time.Now().Sub(now)
	if elapsed > time.Second {
		t.Fatalf("stop too long: %s", elapsed)
	}
	time.Sleep(time.Second)

	sink1.mutex.Lock()
	defer sink1.mutex.Unlock()
	assert.Equal(t, true, sink1.stopped)

	sink2.mutex.Lock()
	defer sink2.mutex.Unlock()
	assert.Equal(t, true, sink2.stopped)
}
