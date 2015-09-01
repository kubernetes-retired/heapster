package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO Instrumentation? To get statistics?
// TODO Authorization / Authentication ?

// More detailed error

type HawkularClientError struct {
	msg  string
	Code int
}

func (self *HawkularClientError) Error() string {
	return fmt.Sprintf("Hawkular returned status code %d, error message: %s", self.Code, self.msg)
}

// Client creation and instance config

const (
	base_url string        = "hawkular/metrics"
	timeout  time.Duration = time.Duration(30 * time.Second)
)

type Parameters struct {
	Tenant string // Technically optional, but requires setting Tenant() option everytime
	Host   string
	Path   string // Modifieral
}

type Client struct {
	Tenant string
	url    *url.URL
	client *http.Client
}

type HawkularClient interface {
	Send(*http.Request) (*http.Response, error)
}

// Modifiers

type Modifier func(*http.Request) error

// Override function to replace the Tenant (defaults to Client default)
func Tenant(tenant string) Modifier {
	return func(r *http.Request) error {
		r.Header.Set("Hawkular-Tenant", tenant)
		return nil
	}
}

// Add payload to the request
func Data(data interface{}) Modifier {
	return func(r *http.Request) error {
		jsonb, err := json.Marshal(data)
		if err != nil {
			return err
		}

		b := bytes.NewBuffer(jsonb)
		rc := ioutil.NopCloser(b)
		r.Body = rc

		// fmt.Printf("Sending: %s\n", string(jsonb))

		if b != nil {
			r.ContentLength = int64(b.Len())
		}
		return nil
	}
}

func (self *Client) Url(method string, e ...Endpoint) Modifier {
	// TODO Create composite URLs? Add().Add().. etc? Easier to modify on the fly..
	return func(r *http.Request) error {
		u := self.createUrl(e...)
		r.URL = u
		r.Method = method
		return nil
	}
}

// Filters for querying

type Filter func(r *http.Request)

func Filters(f ...Filter) Modifier {
	return func(r *http.Request) error {
		for _, filter := range f {
			filter(r)
		}
		return nil // Or should filter return err?
	}
}

// Add query parameters
func Param(k string, v string) Filter {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set(k, v)
		r.URL.RawQuery = q.Encode()
	}
}

func TypeFilter(t MetricType) Filter {
	return Param("type", t.shortForm())
}

func TagsFilter(t map[string]string) Filter {
	j := tagsEncoder(t)
	return Param("tags", j)
}

// Requires HWKMETRICS-233
func IdFilter(regexp string) Filter {
	return Param("id", regexp)
}

func StartTimeFilter(duration time.Duration) Filter {
	return Param("start", strconv.Itoa(int(duration)))
}

func EndTimeFilter(duration time.Duration) Filter {
	return Param("end", strconv.Itoa(int(duration)))
}

func BucketsFilter(buckets int) Filter {
	return Param("buckets", strconv.Itoa(buckets))
}

// The SEND method..

func (self *Client) createRequest() *http.Request {
	req := &http.Request{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       self.url.Host,
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Hawkular-Tenant", self.Tenant)
	return req
}

func (self *Client) Send(o ...Modifier) (*http.Response, error) {
	// Initialize
	r := self.createRequest()

	// Run all the modifiers
	for _, f := range o {
		err := f(r)
		if err != nil {
			return nil, err
		}
	}

	return self.client.Do(r)
}

// Commands

func prepend(slice []Modifier, a ...Modifier) []Modifier {
	p := make([]Modifier, 0, len(slice)+len(a))
	p = append(p, a...)
	p = append(p, slice...)
	return p
}

// Create new Definition
func (self *Client) Create(md MetricDefinition, o ...Modifier) (bool, error) {
	// Keep the order, add custom prepend
	o = prepend(o, self.Url("POST", TypeEndpoint(md.Type)), Data(md))

	r, err := self.Send(o...)
	if err != nil {
		return false, err
	}

	defer r.Body.Close()

	if r.StatusCode > 399 {
		err = self.parseErrorResponse(r)
		if err, ok := err.(*HawkularClientError); ok {
			if err.Code != http.StatusConflict {
				return false, err
			} else {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// Fetch definitions
func (self *Client) Definitions(o ...Modifier) ([]*MetricDefinition, error) {
	o = prepend(o, self.Url("GET", TypeEndpoint(Generic)))

	r, err := self.Send(o...)
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
		return nil, self.parseErrorResponse(r)
	}

	return nil, nil
}

// Update tags
func (self *Client) UpdateTags(t MetricType, id string, tags map[string]string, o ...Modifier) error {
	o = prepend(o, self.Url("PUT", TypeEndpoint(t), SingleMetricEndpoint(id), TagEndpoint()), Data(tags))

	r, err := self.Send(o...)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	if r.StatusCode > 399 {
		return self.parseErrorResponse(r)
	}

	return nil
}

// Delete given tags from the definition
func (self *Client) DeleteTags(t MetricType, id string, tags map[string]string, o ...Modifier) error {
	o = prepend(o, self.Url("DELETE", TypeEndpoint(t), SingleMetricEndpoint(id), TagEndpoint(), TagsEndpoint(tags)))

	r, err := self.Send(o...)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	if r.StatusCode > 399 {
		return self.parseErrorResponse(r)
	}

	return nil
}

// Fetch metric definition tags
func (self *Client) Tags(t MetricType, id string, o ...Modifier) (map[string]string, error) {
	o = prepend(o, self.Url("GET", TypeEndpoint(t), SingleMetricEndpoint(id), TagEndpoint()))

	r, err := self.Send(o...)
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
		return nil, self.parseErrorResponse(r)
	}

	return nil, nil
}

// Write datapoints to the server
func (self *Client) Write(metrics []MetricHeader, o ...Modifier) error {
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
				on = prepend(on, self.Url("POST", TypeEndpoint(k), DataEndpoint()), Data(v))

				r, err := self.Send(on...)
				if err != nil {
					errorsChan <- err
					return
				}

				defer r.Body.Close()

				if r.StatusCode > 399 {
					errorsChan <- self.parseErrorResponse(r)
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

// Read data from the server
func (self *Client) ReadMetric(t MetricType, id string, o ...Modifier) ([]*Datapoint, error) {
	o = prepend(o, self.Url("GET", TypeEndpoint(t), SingleMetricEndpoint(id), DataEndpoint()))

	r, err := self.Send(o...)
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
		dp := []*Datapoint{}
		if b != nil {
			if err = json.Unmarshal(b, &dp); err != nil {
				return nil, err
			}
		}
		return dp, nil
	} else if r.StatusCode > 399 {
		return nil, self.parseErrorResponse(r)
	}

	return nil, nil
}

// Initialization

func NewHawkularClient(p Parameters) (*Client, error) {
	if p.Path == "" {
		p.Path = base_url
	}

	u := &url.URL{
		Host:   p.Host,
		Path:   p.Path,
		Scheme: "http",
		Opaque: fmt.Sprintf("//%s/%s", p.Host, p.Path),
	}
	return &Client{
		url:    u,
		Tenant: p.Tenant,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Public functions

// Older functions..

// Return a single definition
func (self *Client) Definition(t MetricType, id string) (*MetricDefinition, error) {
	url := self.singleMetricsUrl(t, id)

	b, err := self.process(url, "GET", nil)
	if err != nil {
		return nil, err
	}

	md := MetricDefinition{}
	if b != nil {
		if err = json.Unmarshal(b, &md); err != nil {
			return nil, err
		}
	}
	return &md, nil
}

// Read single Gauge metric's datapoints.
// TODO: Remove and replace with better Read properties? Perhaps with iterators?
func (self *Client) SingleGaugeMetric(id string, options map[string]string) ([]*Datapoint, error) {
	id = cleanId(id)
	u := self.paramUrl(self.dataUrl(self.singleMetricsUrl(Gauge, id)), options)

	// fmt.Printf("Receiving for %s, from: %s\n", self.Tenant, u)

	b, err := self.process(u, "GET", nil)
	if err != nil {
		return nil, err
	}
	metrics := []*Datapoint{}

	if b != nil {
		// fmt.Printf("Received: %s\n", string(b))
		if err = json.Unmarshal(b, &metrics); err != nil {
			return nil, err
		}
	}
	return metrics, nil

}

// HTTP Helper functions

func cleanId(id string) string {
	return url.QueryEscape(id)
}

// Override default http.NewRequest to avoid url.Parse which has a bug (removes valid %2F)
func (self *Client) newRequest(url *url.URL, method string, body io.Reader) (*http.Request, error) {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
	}

	req := &http.Request{
		Method:     method,
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       rc,
		Host:       url.Host,
	}

	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
		}
	}
	return req, nil
}

// Helper function that transforms struct to json and fetches the correct tenant information
// TODO: Try the decorator pattern to replace all these simple functions?
func (self *Client) process(url *url.URL, method string, data interface{}) ([]byte, error) {
	jsonb, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}
	return self.send(url, method, jsonb)
}

func (self *Client) send(url *url.URL, method string, json []byte) ([]byte, error) {
	// Have to replicate http.NewRequest here to avoid calling of url.Parse,
	// which has a bug when it comes to encoded url
	req, _ := self.newRequest(url, method, bytes.NewBuffer(json))
	req.Header.Add("Content-Type", "application/json")
	// if len(tenant) > 0 {
	// req.Header.Add("Hawkular-Tenant", tenant)
	// } else {
	req.Header.Add("Hawkular-Tenant", self.Tenant)
	// }

	// fmt.Printf("curl -X %s -H 'Hawkular-Tenant: %s' %s\n", req.Method, req.Header.Get("Hawkular-Tenant"), req.URL)

	resp, err := self.client.Do(req)

	// fmt.Printf("%s\n", resp.Header.Get("Content-Length"))
	// fmt.Printf("%d\n", resp.StatusCode)

	if err != nil {
		return nil, err
	}

	// fmt.Printf("Received bytes: %d\n", resp.ContentLength)

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		return b, err
	} else if resp.StatusCode > 399 {
		return nil, self.parseErrorResponse(resp)
	} else {
		return nil, nil // Nothing to answer..
	}
}

func (self *Client) parseErrorResponse(resp *http.Response) error {
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

// URL functions (...)

type Endpoint func(u *url.URL)

func (self *Client) createUrl(e ...Endpoint) *url.URL {
	mu := *self.url
	for _, f := range e {
		f(&mu)
	}
	return &mu
}

func TypeEndpoint(t MetricType) Endpoint {
	return func(u *url.URL) {
		addToUrl(u, t.String())
	}
}

func SingleMetricEndpoint(id string) Endpoint {
	return func(u *url.URL) {
		addToUrl(u, url.QueryEscape(id))
	}
}

func TagEndpoint() Endpoint {
	return func(u *url.URL) {
		addToUrl(u, "tags")
	}
}

func TagsEndpoint(tags map[string]string) Endpoint {
	return func(u *url.URL) {
		addToUrl(u, tagsEncoder(tags))
	}
}

func DataEndpoint() Endpoint {
	return func(u *url.URL) {
		addToUrl(u, "data")
	}
}

func (self *Client) metricsUrl(metricType MetricType) *url.URL {
	mu := *self.url
	addToUrl(&mu, metricType.String())
	return &mu
}

func (self *Client) singleMetricsUrl(metricType MetricType, id string) *url.URL {
	mu := self.metricsUrl(metricType)
	addToUrl(mu, id)
	return mu
}

func (self *Client) tagsUrl(mt MetricType, id string) *url.URL {
	mu := self.singleMetricsUrl(mt, id)
	addToUrl(mu, "tags")
	return mu
}

func (self *Client) dataUrl(url *url.URL) *url.URL {
	addToUrl(url, "data")
	return url
}

func addToUrl(u *url.URL, s string) *url.URL {
	u.Opaque = fmt.Sprintf("%s/%s", u.Opaque, s)
	return u
}

func tagsEncoder(t map[string]string) string {
	tags := make([]string, 0, len(t))
	for k, v := range t {
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}
	j := strings.Join(tags, ",")
	return j
}

func (self *Client) paramUrl(u *url.URL, options map[string]string) *url.URL {
	q := u.Query()
	for k, v := range options {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()
	return u
}
