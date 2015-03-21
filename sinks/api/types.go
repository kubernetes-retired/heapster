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

package api

import (
	"time"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	cadvisor "github.com/google/cadvisor/info/v1"
)

type MetricType int

const (
	// A cumulative metric.
	MetricCumulative MetricType = iota

	// An instantaneous value metric.
	MetricGauge
)

func (self MetricType) String() string {
	switch self {
	case MetricCumulative:
		return "cumulative"
	case MetricGauge:
		return "gauge"
	}
	return ""
}

type MetricValueType int

const (
	// An int64 value.
	ValueInt64 MetricValueType = iota
	// A boolean value
	ValueBool
	// A double-precision floating point number.
	ValueDouble
)

func (self MetricValueType) String() string {
	switch self {
	case ValueInt64:
		return "int64"
	case ValueBool:
		return "bool"
	case ValueDouble:
		return "double"
	}
	return ""
}

type MetricUnitsType int

const (
	// A counter metric.
	UnitsCount MetricUnitsType = iota
	// A metric in bytes.
	UnitsBytes
	// A metric in milliseconds.
	UnitsMilliseconds
	// A metric in nanoseconds.
	UnitsNanoseconds
)

func (self *MetricUnitsType) String() string {
	switch *self {
	case UnitsBytes:
		return "bytes"
	case UnitsMilliseconds:
		return "ms"
	case UnitsNanoseconds:
		return "ns"
	}
	return ""
}

type LabelDescriptor struct {
	// Key to use for the label.
	Key string `json:"key,omitempty"`

	// Description of the label.
	Description string `json:"description,omitempty"`
}

// TODO: Add cluster name.
type MetricDescriptor struct {
	// The unique name of the metric.
	Name string

	// Description of the metric.
	Description string

	// Descriptor of the labels used by this metric.
	Labels []LabelDescriptor

	// Type and value of metric data.
	Type      MetricType
	ValueType MetricValueType
	Units     MetricUnitsType
}

type Point struct {
	// The name of the metric. Must match an existing descriptor.
	Name string

	// The labels and values for the metric. The keys must match those in the descriptor.
	Labels map[string]string

	// The start and end time for which this data is representative.
	Start time.Time
	End   time.Time

	// The value of the metric. Must match the type in the descriptor.
	Value interface{}
}

// SupportedStatMetric represents a resource usage stat metric.
type SupportedStatMetric struct {
	MetricDescriptor

	// Returns whether this metric is present.
	HasValue func(*cadvisor.ContainerSpec) bool

	// Returns the desired data point for this metric from the stats.
	GetValue func(*cadvisor.ContainerSpec, *cadvisor.ContainerStats) interface{}
}

// Timeseries represents a single metric.
type Timeseries struct {
	Point            *Point
	MetricDescriptor *MetricDescriptor
}

// ExternalSink is the interface that needs to be implemented by all external storage backends.
type ExternalSink interface {
	// Registers a metric with the backend.
	Register([]MetricDescriptor) error
	// Stores input data into the backend.
	// Support input types are as follows:
	// 1. []Timeseries
	// TODO: Add map[string]string to support storing raw data like node or pod status, spec, etc.
	// TODO: Add events.
	StoreTimeseries([]Timeseries) error
	// Returns debug information specific to the sink.
	DebugInfo() string
}

// Codec represents an engine that translated from sources.api to sink.api objects.
type Decoder interface {
	// Timeseries returns the metrics found in input as a timeseries slice.
	Timeseries(input source_api.AggregateData) ([]Timeseries, error)
	// TODO: Process events.
	// TODO: Process config data.
}
