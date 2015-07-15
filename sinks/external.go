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
	"bytes"
	"fmt"
	"strings"
	"sync"

	sink_api_old "github.com/GoogleCloudPlatform/heapster/sinks/api"
	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/golang/glog"
)

type externalSinkManager struct {
	decoder       sink_api_old.Decoder
	externalSinks []sink_api.ExternalSink
	sinkMutex     sync.RWMutex // Protects externalSinks
}

func supportedMetricsDescriptors() []sink_api.MetricDescriptor {
	// Get supported metrics.
	supportedMetrics := sink_api.SupportedStatMetrics()
	for i := range supportedMetrics {
		supportedMetrics[i].Labels = sink_api.SupportedLabels()
	}

	// Create the metrics.
	descriptors := make([]sink_api.MetricDescriptor, 0, len(supportedMetrics))
	for _, supported := range supportedMetrics {
		descriptors = append(descriptors, supported.MetricDescriptor)
	}
	return descriptors
}

// NewExternalSinkManager returns an external sink manager that will manage pushing data to all
// the sinks in 'externalSinks', which is a map of sink name to ExternalSink object.
func NewExternalSinkManager(externalSinks []sink_api.ExternalSink) (ExternalSinkManager, error) {
	m := &externalSinkManager{
		decoder: sink_api_old.NewDecoder(),
	}
	if externalSinks != nil {
		if err := m.SetSinks(externalSinks); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (self *externalSinkManager) Store(input interface{}) error {
	data, ok := input.(source_api.AggregateData)
	if !ok {
		return fmt.Errorf("unknown input type %T", input)
	}
	// TODO: Store data in cache.
	timeseries, err := self.decoder.Timeseries(data)
	if err != nil {
		return err
	}
	// Format metrics and push them.
	self.sinkMutex.RLock()
	defer self.sinkMutex.RUnlock()
	errorsLen := 2 * len(self.externalSinks)
	errorsChan := make(chan error, errorsLen)
	for idx := range self.externalSinks {
		sink := self.externalSinks[idx]
		go func() {
			glog.V(2).Infof("Storing Timeseries to %q", sink.Name())
			errorsChan <- sink.StoreTimeseries(timeseries)
		}()
		go func() {
			glog.V(2).Infof("Storing Events to %q", sink.Name())
			errorsChan <- sink.StoreEvents(data.Events)
		}()
	}
	var errors []string
	for i := 0; i < errorsLen; i++ {
		if err := <-errorsChan; err != nil {
			errors = append(errors, fmt.Sprintf("%v ", err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("encountered the following errors: %s", strings.Join(errors, ";\n"))
	}
	return nil
}

func (self *externalSinkManager) DebugInfo() string {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "External Sinks")

	// Add metrics being exported.
	fmt.Fprintln(b, "\tExported metrics:")
	for _, supported := range sink_api.SupportedStatMetrics() {
		fmt.Fprintf(b, "\t\t%s: %s\n", supported.Name, supported.Description)
	}

	// Add labels being used.
	fmt.Fprintln(b, "\tExported labels:")
	for _, label := range sink_api.SupportedLabels() {
		fmt.Fprintf(b, "\t\t%s: %s\n", label.Key, label.Description)
	}
	fmt.Fprintln(b, "\tExternal Sinks:")
	self.sinkMutex.RLock()
	defer self.sinkMutex.RUnlock()
	for _, externalSink := range self.externalSinks {
		fmt.Fprintf(b, "\t\t%s\n", externalSink.DebugInfo())
	}

	return b.String()
}

// inSlice returns whether an external sink is part of a set (list) of sinks
func inSlice(sink sink_api.ExternalSink, sinks []sink_api.ExternalSink) bool {
	for _, s := range sinks {
		if sink == s {
			return true
		}
	}
	return false
}

func (self *externalSinkManager) SetSinks(newSinks []sink_api.ExternalSink) error {
	self.sinkMutex.Lock()
	defer self.sinkMutex.Unlock()
	oldSinks := self.externalSinks
	descriptors := supportedMetricsDescriptors()
	for _, sink := range oldSinks {
		if inSlice(sink, newSinks) {
			continue
		}
		if err := sink.Unregister(descriptors); err != nil {
			return err
		}
	}
	for _, sink := range newSinks {
		if inSlice(sink, oldSinks) {
			continue
		}
		if err := sink.Register(descriptors); err != nil {
			return err
		}
	}
	self.externalSinks = newSinks
	return nil
}
