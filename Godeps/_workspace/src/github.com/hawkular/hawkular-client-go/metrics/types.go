package metrics

import (
	"fmt"
)

// MetricType restrictions
type MetricType int

const (
	Gauge = iota
	Availability
	Counter
	Generic
)

var longForm = []string{
	"gauges",
	"availability",
	"counter",
	"metrics",
}

var shortForm = []string{
	"gauge",
	"availability",
	"counter",
	"metrics",
}

func (self MetricType) validate() error {
	if int(self) > len(longForm) && int(self) > len(shortForm) {
		return fmt.Errorf("Given MetricType value %d is not valid", self)
	}
	return nil
}

func (self MetricType) String() string {
	if err := self.validate(); err != nil {
		return "unknown"
	}
	return longForm[self]
}

func (self MetricType) shortForm() string {
	if err := self.validate(); err != nil {
		return "unknown"
	}
	return shortForm[self]
}

// Hawkular-Metrics external structs

type MetricHeader struct {
	Type MetricType  `json:"-"`
	Id   string      `json:"id"`
	Data []Datapoint `json:"data"`
}

// Value should be convertible to float64 for numeric values
// Timestamp is milliseconds since epoch
type Datapoint struct {
	Timestamp int64             `json:"timestamp"`
	Value     interface{}       `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
}

type HawkularError struct {
	ErrorMsg string `json:"errorMsg"`
}

type MetricDefinition struct {
	Type          MetricType        `json:"-"`
	Id            string            `json:"id"`
	Tags          map[string]string `json:"tags,omitempty"`
	RetentionTime int               `json:"dataRetention,omitempty"`
}
