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
	"flag"
	"fmt"

	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api"
	"github.com/GoogleCloudPlatform/heapster/sinks/gcm"
	"github.com/GoogleCloudPlatform/heapster/sinks/influxdb"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
)

var (
	argSink = flag.String("sink", "memory", "Backend storage. Options are [memory | influxdb | bigquery | gcm ]")
	// TODO: Take in auth via some other secure mechanism.
	argDbUsername   = flag.String("sink_influxdb_username", "root", "InfluxDB username")
	argDbPassword   = flag.String("sink_influxdb_password", "root", "InfluxDB password")
	argDbHost       = flag.String("sink_influxdb_host", "localhost:8086", "InfluxDB host:port")
	argDbName       = flag.String("sink_influxdb_name", "k8s", "Influxdb database name")
	argAvoidColumns = flag.Bool("sink_influxdb_no_columns", false, "When true, prefixes metric series names with metadata instead of storing metadata in additional columns. Metadata includes hostname, container name, etc. ")
)

type externalSinkManager struct {
	decoder       sink_api.Decoder
	externalSinks []sink_api.ExternalSink
}

func newExternalSinkManager(externalSinks []sink_api.ExternalSink) (ExternalSinkManager, error) {
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
	decoder := sink_api.NewDecoder()
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
	timeseries, err := self.decoder.Timeseries(data)
	if err != nil {
		return err
	}
	// Format metrics and push them.
	var errors []error
	for _, externalSink := range self.externalSinks {
		if err := externalSink.StoreTimeseries(timeseries); err != nil {
			errors = append(errors, err)
		}
	}
	err = nil
	if len(errors) > 0 {
		errStr := ""
		for _, err := range errors {
			errStr = fmt.Sprintf("%v ", err)
		}
		err = fmt.Errorf("encountered the following errors: %s", errStr)
	}

	return err
}

func (self *externalSinkManager) DebugInfo() string {
	desc := "External Sinks\n"

	// Add metrics being exported.
	desc += "\tExported metrics:"
	for _, supported := range sink_api.SupportedStatMetrics() {
		desc += fmt.Sprintf("\t\t%s: %s", supported.Name, supported.Description)
	}

	// Add labels being used.
	desc += "\tExported labels:"
	for _, label := range sink_api.SupportedLabels() {
		desc += fmt.Sprintf("\t\t%s: %s", label.Key, label.Description)
	}
	desc += "\n\tExternal Sinks:"
	for _, externalSink := range self.externalSinks {
		desc += fmt.Sprintf("\n\t\t%s", externalSink.DebugInfo())
	}

	return desc
}

func NewSink() (ExternalSinkManager, error) {
	switch *argSink {
	case "memory":
		return NewMemorySink(), nil
	case "influxdb":
		if *argDbHost == "" {
			return nil, fmt.Errorf("flag '-sink_influxdb_host' invalid")
		}
		if *argDbName == "" {
			return nil, fmt.Errorf("flag '-sink_influxdb_name' invalid")
		}

		externalSink, err := influxdb.NewSink(*argDbHost, *argDbUsername, *argDbPassword, *argDbName, *argAvoidColumns)
		if err != nil {
			return nil, err
		}
		return newExternalSinkManager([]sink_api.ExternalSink{externalSink})
	case "gcm":
		externalSink, err := gcm.NewSink()
		if err != nil {
			return nil, err
		}
		return newExternalSinkManager([]sink_api.ExternalSink{externalSink})
	case "bigquery":
		return NewBigQuerySink()
	default:
		return nil, fmt.Errorf("invalid sink specified - %s", *argSink)
	}
}
