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

type TimePrecision string

const (
	Second      TimePrecision = "s"
	Millisecond TimePrecision = "m"
	Microsecond TimePrecision = "u"
)

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
	// The label keys and values for the metric.
	Labels map[string]string

	// The start and end time for which this data is representative and the precision of the timestamps.
	Start time.Time
	End   time.Time

	// The value of the metric.
	Value interface{}
}

// internalPoint is an internal object.
type internalPoint struct {
	// Overrides any default labels generated for every Point.
	// This is typically used for metric specific labels like 'resource_id'.
	labels map[string]string
	value  interface{}
}

// SupportedStatMetric represents a resource usage stat metric.
type SupportedStatMetric struct {
	MetricDescriptor

	// Returns whether this metric is present.
	HasValue func(*cadvisor.ContainerSpec) bool

	// Returns a slice of internal point objects that contain metric values and associated labels.
	GetValue func(*cadvisor.ContainerSpec, *cadvisor.ContainerStats) []internalPoint
}

// Timeseries represents a set of points for a series.
type Timeseries struct {
	// The name of the series.
	SeriesName string

	// Precision of points timestamps.
	TimePrecision TimePrecision

	// The labels used by this series.
	LabelKeys []string

	// The points to add to this series.
	Points []Point
}

// ExternalSink is the interface that needs to be implemented by all external storage backends.
type ExternalSink interface {
	// Registers a metric with the backend.
	Register([]MetricDescriptor) error
	// Stores input data into the backend. This is an array of Timeseries array pointers so
	// that Timeseries arrays from multiple decoders can be combined with append without being
	// copied.
	StoreTimeseries([]*[]Timeseries) error
	// Returns debug information specific to the sink.
	DebugInfo() string
	// Returns true if this sink supports metrics
	SupportsMetrics() bool
	// Returns true if this sink supports events
	SupportsEvents() bool
}

// Codec represents an engine that translated from sources.api to sink.api objects.
type Decoder interface {
	// Timeseries returns the metrics found in input as a timeseries slice.
	Timeseries(input source_api.AggregateData) ([]Timeseries, error)
	// TODO: Process events.
	// TODO: Process config data.
}
