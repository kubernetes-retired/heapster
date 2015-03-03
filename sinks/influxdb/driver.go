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

package influxdb

import (
	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api"
)

type influxdbSink struct {
}

func (self *influxdbSink) Register(metrics []sink_api.MetricDescriptor) error {
	// Create tags once influxDB v0.9.0 is released.
	return nil
}

func (self *influxdbSink) StoreTimeseries(input []sink_api.Timeseries) error {
	return nil
}

func (self *influxdbSink) DebugInfo() string {
	return "Sink Type: influxDB"
}

// Returns a thread-compatible implementation of influxdb interactions.
func NewInfluxdbExternalSink() (sink_api.ExternalSink, error) {
	return &influxdbSink{}, nil
}
