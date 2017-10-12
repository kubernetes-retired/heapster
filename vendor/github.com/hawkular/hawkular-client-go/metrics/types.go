/*
   Copyright 2015-2016 Red Hat, Inc. and/or its affiliates
   and other contributors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package metrics

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// HawkularClientError Extracted error information from Hawkular-Metrics server
type HawkularClientError struct {
	msg  string
	Code int
}

// Parameters is a struct used as initialization parameters to the client
type Parameters struct {
	Tenant      string // Technically optional, but requires setting Tenant() option everytime
	Url         string
	TLSConfig   *tls.Config
	Token       string
	Concurrency int
}

// Client is HawkularClient's internal data structure
type Client struct {
	Tenant string
	url    *url.URL
	client *http.Client
	Token  string
	pool   chan (*poolRequest)
}

type poolRequest struct {
	req   *http.Request
	rChan chan (*poolResponse)
}

type poolResponse struct {
	err  error
	resp *http.Response
}

// HawkularClient is a base type to define available functions of the client
type HawkularClient interface {
	Send(*http.Request) (*http.Response, error)
}

// Modifier Modifiers base type
type Modifier func(*http.Request) error

// Filter Filter type for querying
type Filter func(r *http.Request)

// Endpoint Endpoint type to define request URL
type Endpoint func(u *url.URL)

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
	"counters",
	"metrics",
}

var shortForm = []string{
	"gauge",
	"availability",
	"counter",
	"metrics",
}

func (mt MetricType) validate() error {
	if int(mt) > len(longForm) && int(mt) > len(shortForm) {
		return fmt.Errorf("Given MetricType value %d is not valid", mt)
	}
	return nil
}

// String is a convenience function to return string representation of type
func (mt MetricType) String() string {
	if err := mt.validate(); err != nil {
		return "unknown"
	}
	return longForm[mt]
}

func (mt MetricType) shortForm() string {
	if err := mt.validate(); err != nil {
		return "unknown"
	}
	return shortForm[mt]
}

// UnmarshalJSON is a custom unmarshaller for MetricType (from string representation)
func (mt *MetricType) UnmarshalJSON(b []byte) error {
	var f interface{}
	err := json.Unmarshal(b, &f)
	if err != nil {
		return err
	}

	if str, ok := f.(string); ok {
		for i, v := range shortForm {
			if str == v {
				*mt = MetricType(i)
				break
			}
		}
	}

	return nil
}

// MarshalJSON is a custom marshaller for MetricType (using string representation)
func (mt MetricType) MarshalJSON() ([]byte, error) {
	return json.Marshal(mt.String())
}

// MetricHeader is the header struct for time series, which has identifiers (tenant, type, id) for uniqueness
// and []Datapoint to describe the actual time series values.
type MetricHeader struct {
	Tenant string      `json:"-"`
	Type   MetricType  `json:"-"`
	ID     string      `json:"id"`
	Data   []Datapoint `json:"data"`
}

// Datapoint is a struct that represents a single time series value.
// Value should be convertible to float64 for gauge/counter series.
// Timestamp accuracy is milliseconds since epoch
type Datapoint struct {
	Timestamp time.Time         `json:"-"`
	Value     interface{}       `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// MarshalJSON is modified JSON marshalling for Datapoint object to modify time.Time to milliseconds since epoch
func (d Datapoint) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(map[string]interface{}{
		"timestamp": ToUnixMilli(d.Timestamp),
		"value":     d.Value,
	})

	return b, err
}

// To avoid recursion in UnmarshalJSON
type datapoint Datapoint

type datapointJSON struct {
	datapoint
	Ts int64 `json:"timestamp"`
}

// UnmarshalJSON is a custom unmarshaller for Datapoint for timestamp modifications
func (d *Datapoint) UnmarshalJSON(b []byte) error {
	dp := datapointJSON{}
	err := json.Unmarshal(b, &dp)
	if err != nil {
		return err
	}

	*d = Datapoint(dp.datapoint)
	d.Timestamp = FromUnixMilli(dp.Ts)

	return nil
}

// HawkularError is the return payload from Hawkular-Metrics if processing failed
type HawkularError struct {
	ErrorMsg string `json:"errorMsg"`
}

// MetricDefinition is a struct that describes the stored definition of a time serie
type MetricDefinition struct {
	Tenant        string            `json:"-"`
	Type          MetricType        `json:"type,omitempty"`
	ID            string            `json:"id"`
	Tags          map[string]string `json:"tags,omitempty"`
	RetentionTime int               `json:"dataRetention,omitempty"`
}

// TODO Fix the Start & End to return a time.Time

// Bucketpoint is a return structure for bucketed data requests (stats endpoint)
type Bucketpoint struct {
	Start       int64        `json:"start"`
	End         int64        `json:"end"`
	Min         float64      `json:"min"`
	Max         float64      `json:"max"`
	Avg         float64      `json:"avg"`
	Median      float64      `json:"median"`
	Empty       bool         `json:"empty"`
	Samples     int64        `json:"samples"`
	Percentiles []Percentile `json:"percentiles"`
}

// Percentile is Hawkular-Metrics' estimated (not exact) percentile
type Percentile struct {
	Quantile float64 `json:"quantile"`
	Value    float64 `json:"value"`
}

// Order is a basetype for selecting the sorting of requested datapoints
type Order int

const (
	// ASC Ascending
	ASC = iota
	// DESC Descending
	DESC
)

// String Get string representation of type
func (o Order) String() string {
	switch o {
	case ASC:
		return "ASC"
	case DESC:
		return "DESC"
	}
	return ""
}
