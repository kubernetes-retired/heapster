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
	"fmt"
	"strings"

	sink_api_old "github.com/GoogleCloudPlatform/heapster/sinks/api"
	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/golang/glog"
)

type externalSinkManager struct {
	decoder       sink_api_old.Decoder
	externalSinks []sink_api.ExternalSink
}

// NewExternalSinkManager returns an external sink manager that will manage pushing data to all
// the sinks in 'externalSinks', which is a map of sink name to ExternalSink object.
func NewExternalSinkManager(externalSinks []sink_api.ExternalSink) (ExternalSinkManager, error) {
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

	for _, externalSink := range externalSinks {
		err := externalSink.Register(descriptors)
		if err != nil {
			return nil, err
		}
	}
	decoder := sink_api_old.NewDecoder()
	return &externalSinkManager{
		externalSinks: externalSinks,
		decoder:       decoder,
	}, nil
}

// TODO(vmarmol): Paralellize this.
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
	errorsChan := make(chan error, len(self.externalSinks))
	for idx := range self.externalSinks {
		sink := self.externalSinks[idx]
		go func(sink sink_api.ExternalSink) {
			glog.V(2).Infof("Storing Timeseries to %q", sink.Name())
			errorsChan <- sink.StoreTimeseries(timeseries)
		}(sink)
		go func(sink sink_api.ExternalSink) {
			glog.V(2).Infof("Storing Events to %q", sink.Name())
			errorsChan <- sink.StoreEvents(data.Events)
		}(sink)
	}
	var errors []string
	for i := 1; i <= len(errorsChan); i++ {
		if err := <-errorsChan; err != nil {
			errors = append(errors, fmt.Sprintf("%v ", err))
		}
	}
	err = nil
	if len(errors) > 0 {
		err = fmt.Errorf("encountered the following errors: %s", strings.Join(errors, ";\n"))
	}

	return err
}

func (self *externalSinkManager) DebugInfo() string {
	desc := "External Sinks\n"

	// Add metrics being exported.
	desc += "\tExported metrics:\n"
	for _, supported := range sink_api.SupportedStatMetrics() {
		desc += fmt.Sprintf("\t\t%s: %s", supported.Name, supported.Description)
	}

	// Add labels being used.
	desc += "\n\tExported labels:\n"
	for _, label := range sink_api.SupportedLabels() {
		desc += fmt.Sprintf("\t\t%s: %s", label.Key, label.Description)
	}
	desc += "\n\tExternal Sinks:\n"
	for _, externalSink := range self.externalSinks {
		desc += fmt.Sprintf("\n\t\t%s", externalSink.DebugInfo())
	}

	return desc
}
