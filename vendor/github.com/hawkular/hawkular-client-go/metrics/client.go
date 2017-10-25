/*
   Copyright 2015-2017 Red Hat, Inc. and/or its affiliates
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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (c *HawkularClientError) Error() string {
	return fmt.Sprintf("Hawkular returned status code %d, error message: %s", c.Code, c.msg)
}

// Client creation and instance config

const (
	baseURL            string        = "hawkular/metrics"
	defaultConcurrency int           = 1
	timeout            time.Duration = time.Duration(30 * time.Second)
	tenantHeader       string        = "Hawkular-Tenant"
	adminHeader        string        = "Hawkular-Admin-Token"
)

// Tenant function replaces the Tenant in the request (instead of using the default in Client parameters)
func Tenant(tenant string) Modifier {
	return func(r *http.Request) error {
		r.Header.Set(tenantHeader, tenant)
		return nil
	}
}

// AdminAuthentication function to add metrics' admin token to the request
func AdminAuthentication(token string) Modifier {
	return func(r *http.Request) error {
		r.Header.Add(adminHeader, token)
		return nil
	}
}

// Data adds payload to the request
func Data(data interface{}) Modifier {
	return func(r *http.Request) error {
		jsonb, err := json.Marshal(data)
		if err != nil {
			return err
		}

		b := bytes.NewBuffer(jsonb)
		rc := ioutil.NopCloser(b)
		r.Body = rc

		if b != nil {
			r.ContentLength = int64(b.Len())
		}
		return nil
	}
}

// URL sets the request URL
func (c *Client) URL(method string, e ...Endpoint) Modifier {
	// TODO Create composite URLs? Add().Add().. etc? Easier to modify on the fly..
	// And also remove the necessary order of Adds
	return func(r *http.Request) error {
		u := c.createURL(e...)
		r.URL = u
		r.Method = method
		return nil
	}
}

// Filters allows using multiple Filter types in the same request
func Filters(f ...Filter) Modifier {
	return func(r *http.Request) error {
		for _, filter := range f {
			filter(r)
		}
		return nil // Or should filter return err?
	}
}

// Param adds query parameters to the request
func Param(k string, v string) Filter {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set(k, v)
		r.URL.RawQuery = q.Encode()
	}
}

// TypeFilter is a query parameter to filter by type
func TypeFilter(t MetricType) Filter {
	return Param("type", fmt.Sprint(t))
}

// TagsFilter is a query parameter to filter with tags query
func TagsFilter(t map[string]string) Filter {
	j := tagsEncoder(t, false)
	return Param("tags", j)
}

// TagsQueryFilter is a query parameter for the new style tags query language
func TagsQueryFilter(query ...string) Filter {
	tagQl := strings.Join(query, " AND ")
	return Param("tags", tagQl)
}

// IdFilter is a query parameter to add filtering by id name
func IdFilter(regexp string) Filter {
	return Param("id", regexp)
}

// StartTimeFilter is a query parameter to filter with start time
func StartTimeFilter(startTime time.Time) Filter {
	// return Param("start", strconv.Itoa(int(startTime.Unix())))
	return Param("start", strconv.Itoa(int(ToUnixMilli(startTime))))
}

// EndTimeFilter is a query parameter to filter with end time
func EndTimeFilter(endTime time.Time) Filter {
	return Param("end", strconv.Itoa(int(ToUnixMilli(endTime))))
}

// BucketsFilter is a query parameter to define amount of buckets
func BucketsFilter(buckets int) Filter {
	return Param("buckets", strconv.Itoa(buckets))
}

// BucketsDurationFilter is a query parameter to set the size of a bucket based on duration
// Minimum supported bucket is 1 millisecond
func BucketsDurationFilter(duration time.Duration) Filter {
	return Param("bucketDuration", fmt.Sprintf("%dms", (duration.Nanoseconds()/1e6)))
}

// LimitFilter is a query parameter to limit result count
func LimitFilter(limit int) Filter {
	return Param("limit", strconv.Itoa(limit))
}

// OrderFilter Query parameter to define the ordering of datapoints
func OrderFilter(order Order) Filter {
	return Param("order", order.String())
}

// StartFromBeginningFilter returns data from the oldest stored datapoint
func StartFromBeginningFilter() Filter {
	return Param("fromEarliest", "true")
}

// StackedFilter forces downsampling of stacked return values
func StackedFilter() Filter {
	return Param("stacked", "true")
}

// PercentilesFilter is a query parameter to define the requested percentiles
func PercentilesFilter(percentiles []float64) Filter {
	s := make([]string, 0, len(percentiles))
	for _, v := range percentiles {
		s = append(s, fmt.Sprintf("%v", v))
	}
	j := strings.Join(s, ",")
	return Param("percentiles", j)
}

// The SEND method..

func (c *Client) createRequest() *http.Request {
	req := &http.Request{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       c.url.Host,
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(tenantHeader, c.Tenant)

	if len(c.Credentials) > 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", c.Credentials))
	}

	if len(c.Token) > 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	return req
}

// Send sends a constructed request to the Hawkular-Metrics server.
// All the requests are pooled and limited by set concurrency limits
func (c *Client) Send(o ...Modifier) (*http.Response, error) {
	// Initialize
	r := c.createRequest()

	// Run all the modifiers
	for _, f := range o {
		err := f(r)
		if err != nil {
			return nil, err
		}
	}

	rChan := make(chan *poolResponse)
	preq := &poolRequest{r, rChan}

	c.pool <- preq

	presp := <-rChan
	close(rChan)

	return presp.resp, presp.err
}

// Commands

// Tenants returns a list of tenants from the server
func (c *Client) Tenants(o ...Modifier) ([]*TenantDefinition, error) {
	o = prepend(o, c.URL("GET", TenantEndpoint()), AdminAuthentication(c.AdminToken))

	r, err := c.Send(o...)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		tenants := []*TenantDefinition{}
		if b != nil {
			if err = json.Unmarshal(b, &tenants); err != nil {
				return nil, err
			}
		}
		return tenants, err
	} else if r.StatusCode > 399 {
		return nil, c.parseErrorResponse(r)
	}

	return nil, nil
}

// CreateTenant creates a tenant definition on the server
func (c *Client) CreateTenant(tenant TenantDefinition, o ...Modifier) (bool, error) {
	o = prepend(o, c.URL("POST", TenantEndpoint()), AdminAuthentication(c.AdminToken), Data(tenant))

	r, err := c.Send(o...)
	if err != nil {
		return false, err
	}

	defer r.Body.Close()

	if r.StatusCode > 399 {
		err = c.parseErrorResponse(r)
		if err, ok := err.(*HawkularClientError); ok {
			if err.Code != http.StatusConflict {
				return false, err
			}
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create creates a new metric definition
func (c *Client) Create(md MetricDefinition, o ...Modifier) (bool, error) {
	// Keep the order, add custom prepend
	o = prepend(o, c.URL("POST", TypeEndpoint(md.Type)), Data(md))

	r, err := c.Send(o...)
	if err != nil {
		return false, err
	}

	defer r.Body.Close()

	if r.StatusCode > 399 {
		err = c.parseErrorResponse(r)
		if err, ok := err.(*HawkularClientError); ok {
			if err.Code != http.StatusConflict {
				return false, err
			}
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Definitions fetches metric definitions from the server
func (c *Client) Definitions(o ...Modifier) ([]*MetricDefinition, error) {
	o = prepend(o, c.URL("GET", TypeEndpoint(Generic)))

	r, err := c.Send(o...)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		md := []*MetricDefinition{}
		if b != nil {
			if err = json.Unmarshal(b, &md); err != nil {
				return nil, err
			}
		}
		return md, err
	} else if r.StatusCode > 399 {
		return nil, c.parseErrorResponse(r)
	}

	return nil, nil
}

// Definition returns a single metric definition
func (c *Client) Definition(t MetricType, id string, o ...Modifier) (*MetricDefinition, error) {
	o = prepend(o, c.URL("GET", TypeEndpoint(t), SingleMetricEndpoint(id)))

	r, err := c.Send(o...)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		md := &MetricDefinition{}
		if b != nil {
			if err = json.Unmarshal(b, md); err != nil {
				return nil, err
			}
		}
		return md, err
	} else if r.StatusCode > 399 {
		return nil, c.parseErrorResponse(r)
	}

	return nil, nil
}

// TagValues queries for available tagValues
func (c *Client) TagValues(tagQuery map[string]string, o ...Modifier) (map[string][]string, error) {
	o = prepend(o, c.URL("GET", TypeEndpoint(Generic), TagEndpoint(), TagsEndpoint(tagQuery)))

	r, err := c.Send(o...)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		md := make(map[string][]string)
		if b != nil {
			if err = json.Unmarshal(b, &md); err != nil {
				return nil, err
			}
		}
		return md, err
	} else if r.StatusCode > 399 {
		return nil, c.parseErrorResponse(r)
	}

	return nil, nil
}

// UpdateTags modifies the tags of a metric definition
func (c *Client) UpdateTags(t MetricType, id string, tags map[string]string, o ...Modifier) error {
	o = prepend(o, c.URL("PUT", TypeEndpoint(t), SingleMetricEndpoint(id), TagEndpoint()), Data(tags))

	r, err := c.Send(o...)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	if r.StatusCode > 399 {
		return c.parseErrorResponse(r)
	}

	return nil
}

// DeleteTags deletes given tags from the definition
func (c *Client) DeleteTags(t MetricType, id string, tags []string, o ...Modifier) error {
	o = prepend(o, c.URL("DELETE", TypeEndpoint(t), SingleMetricEndpoint(id), TagEndpoint(), TagNamesEndpoint(tags)))

	r, err := c.Send(o...)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	if r.StatusCode > 399 {
		return c.parseErrorResponse(r)
	}

	return nil
}

// Tags fetches metric definition's tags
func (c *Client) Tags(t MetricType, id string, o ...Modifier) (map[string]string, error) {
	o = prepend(o, c.URL("GET", TypeEndpoint(t), SingleMetricEndpoint(id), TagEndpoint()))

	r, err := c.Send(o...)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		tags := make(map[string]string)
		if b != nil {
			if err = json.Unmarshal(b, &tags); err != nil {
				return nil, err
			}
		}
		return tags, nil
	} else if r.StatusCode > 399 {
		return nil, c.parseErrorResponse(r)
	}

	return nil, nil
}

// Write writes datapoints to the server
func (c *Client) Write(metrics []MetricHeader, o ...Modifier) error {
	if len(metrics) > 0 {
		mHs := make(map[MetricType][]MetricHeader)
		for _, m := range metrics {
			if _, found := mHs[m.Type]; !found {
				mHs[m.Type] = make([]MetricHeader, 0, 1)
			}
			mHs[m.Type] = append(mHs[m.Type], m)
		}

		wg := &sync.WaitGroup{}
		errorsChan := make(chan error, len(mHs))

		for k, v := range mHs {
			wg.Add(1)
			go func(k MetricType, v []MetricHeader) {
				defer wg.Done()

				// Should be sorted and splitted by type & tenant..
				on := o
				on = prepend(on, c.URL("POST", TypeEndpoint(k), RawEndpoint()), Data(v))

				r, err := c.Send(on...)
				if err != nil {
					errorsChan <- err
					return
				}

				defer r.Body.Close()

				if r.StatusCode > 399 {
					errorsChan <- c.parseErrorResponse(r)
				}
			}(k, v)
		}
		wg.Wait()
		select {
		case err, ok := <-errorsChan:
			if ok {
				return err
			}
			// If channel is closed, we're done
		default:
			// Nothing to do
		}

	}
	return nil
}

// ReadRaw reads metric datapoints from the server for the given metric
func (c *Client) ReadRaw(t MetricType, id string, o ...Modifier) ([]*Datapoint, error) {
	o = prepend(o, c.URL("GET", TypeEndpoint(t), SingleMetricEndpoint(id), RawEndpoint()))

	r, err := c.Send(o...)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		dp := []*Datapoint{}
		if b != nil {
			if err = json.Unmarshal(b, &dp); err != nil {
				return nil, err
			}
		}
		return dp, nil
	} else if r.StatusCode > 399 {
		return nil, c.parseErrorResponse(r)
	}

	return nil, nil
}

// ReadBuckets reads datapoints from the server, aggregated to buckets with given parameters.
func (c *Client) ReadBuckets(t MetricType, o ...Modifier) ([]*Bucketpoint, error) {
	o = prepend(o, c.URL("GET", TypeEndpoint(t), StatsEndpoint()))

	r, err := c.Send(o...)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		// Check for GaugeBucketpoint and so on for the rest.. uh
		bp := []*Bucketpoint{}
		if b != nil {
			if err = json.Unmarshal(b, &bp); err != nil {
				return nil, err
			}
		}
		return bp, nil
	} else if r.StatusCode > 399 {
		return nil, c.parseErrorResponse(r)
	}

	return nil, nil
}

// NewHawkularClient returns a new initialized instance of client
func NewHawkularClient(p Parameters) (*Client, error) {
	uri, err := url.Parse(p.Url)
	if err != nil {
		return nil, err
	}

	if (p.Username != "" && p.Password == "") || (p.Username == "" && p.Password != "") {
		return nil, fmt.Errorf("To configure credentials, you must specify both Username and Password")
	}

	if (p.Username != "" && p.Password != "") && (p.Token != "") {
		return nil, fmt.Errorf("You cannot specify both Username/Password credentials and a Token.")
	}

	if uri.Path == "" {
		uri.Path = baseURL
	}

	u := &url.URL{
		Host:   uri.Host,
		Path:   uri.Path,
		Scheme: uri.Scheme,
		Opaque: fmt.Sprintf("/%s", uri.Path),
	}

	c := &http.Client{
		Timeout: timeout,
	}
	if p.TLSConfig != nil {
		transport := &http.Transport{TLSClientConfig: p.TLSConfig}
		c.Transport = transport
	}

	var creds string
	if p.Username != "" && p.Password != "" {
		creds = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", p.Username, p.Password)))
	}

	if p.Concurrency < 1 {
		p.Concurrency = 1
	}

	client := &Client{
		url:         u,
		Tenant:      p.Tenant,
		Credentials: creds,
		Token:       p.Token,
		AdminToken:  p.AdminToken,
		client:      c,
		pool:        make(chan *poolRequest, p.Concurrency),
	}

	for i := 0; i < p.Concurrency; i++ {
		go client.sendRoutine()
	}

	return client, nil
}

// Close safely closes the Hawkular-Metrics client and flushes remaining writes to the server
func (c *Client) Close() {
	close(c.pool)
}

// HTTP Helper functions

func (c *Client) parseErrorResponse(resp *http.Response) error {
	// Parse error messages here correctly..
	reply, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &HawkularClientError{Code: resp.StatusCode,
			msg: fmt.Sprintf("Reply could not be read: %s", err.Error()),
		}
	}

	details := &HawkularError{}

	err = json.Unmarshal(reply, details)
	if err != nil {
		return &HawkularClientError{Code: resp.StatusCode,
			msg: fmt.Sprintf("Reply could not be parsed: %s", err.Error()),
		}
	}

	return &HawkularClientError{Code: resp.StatusCode,
		msg: details.ErrorMsg,
	}
}

// Endpoint URL functions (...)

func (c *Client) createURL(e ...Endpoint) *url.URL {
	mu := *c.url
	for _, f := range e {
		f(&mu)
	}
	return &mu
}

// TenantEndpoint is a URL endpoint to fetch tenant related information
func TenantEndpoint() Endpoint {
	return func(u *url.URL) {
		addToURL(u, "tenants")
	}
}

// TypeEndpoint is a URL endpoint setting metricType
func TypeEndpoint(t MetricType) Endpoint {
	return func(u *url.URL) {
		switch t {
		case Gauge:
			addToURL(u, "gauges")
		case Counter:
			addToURL(u, "counters")
		case String:
			addToURL(u, "strings")
		default:
			addToURL(u, string(t))
		}
	}
}

// SingleMetricEndpoint is a URL endpoint for requesting single metricID
func SingleMetricEndpoint(id string) Endpoint {
	return func(u *url.URL) {
		addToURL(u, URLEscape(id))
	}
}

// TagEndpoint is a URL endpoint to check tags information
func TagEndpoint() Endpoint {
	return func(u *url.URL) {
		addToURL(u, "tags")
	}
}

// TagsEndpoint is a URL endpoint which adds tags query
func TagsEndpoint(tags map[string]string) Endpoint {
	return func(u *url.URL) {
		addToURL(u, tagsEncoder(tags, true))
	}
}

// TagNamesEndpoint is a URL endpoint which adds tags names (no values)
func TagNamesEndpoint(tagNames []string) Endpoint {
	return func(u *url.URL) {
		escapedNames := make([]string, 0, len(tagNames))
		for _, v := range tagNames {
			escapedNames = append(escapedNames, URLEscape(v))
		}
		tags := strings.Join(escapedNames, ",")
		addToURL(u, tags)
	}
}

// RawEndpoint is an endpoint to read and write raw datapoints
func RawEndpoint() Endpoint {
	return func(u *url.URL) {
		addToURL(u, "raw")
	}
}

// StatsEndpoint is an endpoint to read aggregated metrics
func StatsEndpoint() Endpoint {
	return func(u *url.URL) {
		addToURL(u, "stats")
	}
}

func addToURL(u *url.URL, s string) *url.URL {
	u.Opaque = fmt.Sprintf("%s/%s", u.Opaque, s)
	return u
}

func tagsEncoder(t map[string]string, escape bool) string {
	tags := make([]string, 0, len(t))
	for k, v := range t {
		if escape {
			k = URLEscape(k)
			v = URLEscape(v)
		}
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}
	j := strings.Join(tags, ",")
	return j
}

// URLEscape Is a fixed version of Golang's URL escaping handling
func URLEscape(input string) string {
	escaped := url.QueryEscape(input)
	escaped = strings.Replace(escaped, "+", "%20", -1)
	return escaped
}
