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

	"github.com/stretchr/testify/assert"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	sink_api "k8s.io/heapster/sinks/api/v1"
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
	return ""
}

func (d *DummySink) Name() string {
	return ""
}

func TestSetSinksRegister(t *testing.T) {
	as := assert.New(t)
	s1 := &DummySink{}
	as.Equal(0, s1.Registered)
	m, err := NewExternalSinkManager([]sink_api.ExternalSink{s1})
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
	m, err := NewExternalSinkManager([]sink_api.ExternalSink{s1})
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
	m, err := NewExternalSinkManager([]sink_api.ExternalSink{s1})
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

func TestSetSinksStore(t *testing.T) {
	as := assert.New(t)
	s1 := &DummySink{}
	m, err := NewExternalSinkManager([]sink_api.ExternalSink{s1})
	as.Nil(err)
	as.Equal(0, s1.StoredTimeseries)
	as.Equal(0, s1.StoredEvents)
	err = m.Store(source_api.AggregateData{})
	as.Nil(err)
	as.Equal(1, s1.StoredTimeseries)
	as.Equal(1, s1.StoredEvents)
	err = m.SetSinks([]sink_api.ExternalSink{})
	as.Nil(err)
	err = m.Store(source_api.AggregateData{})
	as.Nil(err)
	as.Equal(1, s1.StoredTimeseries)
	as.Equal(1, s1.StoredEvents)
	err = m.SetSinks([]sink_api.ExternalSink{s1})
	as.Nil(err)
	err = m.Store(source_api.AggregateData{})
	as.Nil(err)
	as.Equal(2, s1.StoredTimeseries)
	as.Equal(2, s1.StoredEvents)
	err = m.Store(source_api.AggregateData{})
	as.Nil(err)
	as.Equal(3, s1.StoredTimeseries)
	as.Equal(3, s1.StoredEvents)
}
