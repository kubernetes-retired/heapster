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

package core

import (
	"time"
)

type MetricType int8

const (
	MetricCumulative MetricType = iota
	MetricGauge
)

type ValueType int8

const (
	ValueInt64 ValueType = iota
	ValueDouble
)

type MetricValue struct {
	MetricType MetricType
	ValueType  ValueType
	Value      interface{}
}

type MetricSet struct {
	MetricValues map[string]MetricValue
	Labels       map[string]string
}

type DataBatch struct {
	Timestamp time.Time
	// Should use key functions from ms_keys.go
	MetricSets map[string]*MetricSet
}

// A place from where the metrics should be scraped.
type MetricsSource interface {
	ScrapeMetrics(start, end time.Time) *DataBatch
}

type DataSink interface {
	Name() string

	// Exports data to the external storge. The funciton should be synchronous/blocking and finish only
	// after the given DataBatch was written. This will allow sink manager to push data only to these
	// sinks that finished writing the previous data.
	ExportData(*DataBatch)
	Stop()
}
