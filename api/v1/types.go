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

package v1

import (
	"time"
)

// Timeseries represents a set of metrics for the same target object
// (typically a container).
type Timeseries struct {
	// Map of metric names to their values.
	Metrics map[string][]Point `json:"metrics"`

	// Common labels for all metrics.
	Labels map[string]string `json:"labels,omitempty"`
}

// Point represent a metric value.
type Point struct {
	// The start and end time for which this data is representative.
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`

	// Labels specific to this data point.
	Labels map[string]string `json:"labels,omitempty"`

	// The value of the metric.
	Value interface{} `json:"value"`
}
