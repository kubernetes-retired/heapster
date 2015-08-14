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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	fuzz "github.com/google/gofuzz"
	sink_api "k8s.io/heapster/sinks/api"
	"k8s.io/heapster/sinks/cache"
	source_api "k8s.io/heapster/sources/api"
)

type DummySink struct {
	Registered       int
	Unregistered     int
	StoredTimeseries int
	StoredEvents     int
}

func (d *DummySink) Register([]sink_api.MetricDescriptor) error {
	d.Registered++
	return nil
}

func (d *DummySink) Unregister([]sink_api.MetricDescriptor) error {
	d.Unregistered++
	return nil
}

func (d *DummySink) StoreTimeseries([]sink_api.Timeseries) error {
	d.StoredTimeseries++
	return nil
}

func (d *DummySink) StoreEvents([]kube_api.Event) error {
	d.StoredEvents++
	return nil
}

func (d *DummySink) DebugInfo() string {
	return "dummy sink"
}

func (d *DummySink) Name() string {
	return "dummy"
}

func TestSetSinksRegister(t *testing.T) {
	as := assert.New(t)
	s1 := &DummySink{}
	as.Equal(0, s1.Registered)
	m, err := NewExternalSinkManager([]sink_api.ExternalSink{s1}, nil, time.Minute)
	as.Nil(err)
	as.Equal(1, s1.Registered)
	err = m.SetSinks([]sink_api.ExternalSink{s1})
	as.Nil(err)
	s2 := &DummySink{}
	as.Equal(1, s1.Registered)
	as.Equal(0, s2.Registered)
	err = m.SetSinks([]sink_api.ExternalSink{s2})
	as.Nil(err)
	as.Equal(1, s1.Registered)
	as.Equal(1, s2.Registered)
	err = m.SetSinks([]sink_api.ExternalSink{})
	as.Nil(err)
	as.Equal(1, s1.Registered)
	as.Equal(1, s2.Registered)
}

func TestSetSinksUnregister(t *testing.T) {
	as := assert.New(t)
	s1 := &DummySink{}
	as.Equal(0, s1.Unregistered)
	m, err := NewExternalSinkManager([]sink_api.ExternalSink{s1}, nil, time.Minute)
	as.Nil(err)
	as.Equal(0, s1.Unregistered)
	err = m.SetSinks([]sink_api.ExternalSink{s1})
	as.Nil(err)
	s2 := &DummySink{}
	as.Equal(0, s1.Unregistered)
	as.Equal(0, s2.Unregistered)
	err = m.SetSinks([]sink_api.ExternalSink{s2})
	as.Nil(err)
	as.Equal(1, s1.Unregistered)
	as.Equal(0, s2.Unregistered)
	err = m.SetSinks([]sink_api.ExternalSink{})
	as.Nil(err)
	as.Equal(1, s1.Unregistered)
	as.Equal(1, s2.Unregistered)
}

func TestSetSinksRegisterAgain(t *testing.T) {
	as := assert.New(t)
	s1 := &DummySink{}
	as.Equal(0, s1.Registered)
	as.Equal(0, s1.Unregistered)
	m, err := NewExternalSinkManager([]sink_api.ExternalSink{s1}, nil, time.Minute)
	as.Nil(err)
	as.Equal(1, s1.Registered)
	as.Equal(0, s1.Unregistered)
	err = m.SetSinks([]sink_api.ExternalSink{})
	as.Nil(err)
	as.Equal(1, s1.Registered)
	as.Equal(1, s1.Unregistered)
	err = m.SetSinks([]sink_api.ExternalSink{s1})
	as.Nil(err)
	as.Equal(2, s1.Registered)
	as.Equal(1, s1.Unregistered)
	err = m.SetSinks([]sink_api.ExternalSink{})
	as.Nil(err)
	as.Equal(2, s1.Registered)
	as.Equal(2, s1.Unregistered)
}

func newExternalSinkManager(externalSinks []sink_api.ExternalSink, cache cache.Cache, syncFrequency time.Duration) (*externalSinkManager, error) {
	m := &externalSinkManager{
		decoder:       sink_api.NewDecoder(),
		cache:         cache,
		lastSync:      time.Time{},
		syncFrequency: syncFrequency,
	}
	if externalSinks != nil {
		if err := m.SetSinks(externalSinks); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func TestSetSinksStore(t *testing.T) {
	as := assert.New(t)
	s1 := &DummySink{}
	c := cache.NewCache(time.Minute, time.Hour)
	m, err := newExternalSinkManager([]sink_api.ExternalSink{s1}, c, time.Microsecond)
	as.Nil(err)
	as.Equal(0, s1.StoredTimeseries)
	as.Equal(0, s1.StoredEvents)
	var (
		pods       []source_api.Pod
		containers []source_api.Container
		events     []*cache.Event
	)
	f := fuzz.New().NumElements(1, 1).NilChance(0)
	f.Fuzz(&pods)
	f.Fuzz(&containers)
	f.Fuzz(&events)
	c.StorePods(pods)
	c.StoreContainers(containers)
	c.StoreEvents(events)
	m.sync()
	as.Equal(1, s1.StoredTimeseries)
	as.Equal(1, s1.StoredEvents)
	err = m.SetSinks([]sink_api.ExternalSink{})
	as.Nil(err)
	m.sync()
	as.Equal(1, s1.StoredTimeseries)
	as.Equal(1, s1.StoredEvents)

	err = m.SetSinks([]sink_api.ExternalSink{s1})
	as.Equal(1, s1.StoredTimeseries)
	as.Equal(1, s1.StoredEvents)
	as.Nil(err)
	f.Fuzz(&pods)
	f.Fuzz(&containers)
	f.Fuzz(&events)
	c.StorePods(pods)
	c.StoreContainers(containers)
	c.StoreEvents(events)
	m.sync()
	time.Sleep(time.Second)
	as.Equal(2, s1.StoredTimeseries)
	as.Equal(2, s1.StoredEvents)
}
