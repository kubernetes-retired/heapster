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

package monasca

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/golang/glog"
	"k8s.io/heapster/extpoints"
	sinksApi "k8s.io/heapster/sinks/api"
	kube_api "k8s.io/kubernetes/pkg/api"
)

type monascaSink struct {
	client Client
}

// a monasca metric definition
type metric struct {
	Name       string            `json:"name"`
	Dimensions map[string]string `json:"dimensions"`
	Timestamp  int64             `json:"timestamp"`
	Value      float64           `json:"value"`
	ValueMeta  map[string]string `json:"value_meta"`
}

// Pushes the specified metric measurement to the Monasca API.
// The Timeseries are transformed to monasca metrics beforehand.
// Timeseries that cannot be translated to monasca metrics are skipped.
func (sink monascaSink) StoreTimeseries(input []sinksApi.Timeseries) error {
	metrics := sink.processMetrics(input)
	code, response, err := sink.client.SendRequest("POST", "/metrics", metrics)
	if err != nil {
		return err
	}
	if code != http.StatusNoContent {
		return fmt.Errorf(response)
	}
	glog.Infof("metrics pushed: OK")
	return nil
}

func (sink monascaSink) processMetrics(input []sinksApi.Timeseries) []metric {
	metrics := []metric{}
	for _, entry := range input {
		value, err := sink.convertValue(entry.Point.Value)
		if err != nil {
			glog.Warningf("Metric cannot be pushed to monasca. %#v", entry.Point.Value)
		} else {
			dims, valueMeta := sink.processLabels(entry.Point.Labels)
			m := metric{
				Name:       strings.Replace(entry.MetricDescriptor.Name, "/", ".", -1),
				Dimensions: dims,
				Timestamp:  (entry.Point.End.UnixNano() / 1000000),
				Value:      value,
				ValueMeta:  valueMeta,
			}
			metrics = append(metrics, m)
		}
	}
	return metrics
}

// convert the Timeseries value to a monasca value
func (sink monascaSink) convertValue(val interface{}) (float64, error) {
	switch val.(type) {
	case int:
		return float64(val.(int)), nil
	case int64:
		return float64(val.(int64)), nil
	case bool:
		if val.(bool) {
			return 1.0, nil
		}
		return 0.0, nil
	case float32:
		return float64(val.(float32)), nil
	case float64:
		return val.(float64), nil
	}
	return 0.0, fmt.Errorf("Unsupported monasca metric value type %T", reflect.TypeOf(val))
}

const (
	emptyValue       = "none"
	monascaComponent = "component"
	monascaService   = "service"
	monascaHostname  = "hostname"
)

// preprocesses heapster labels, splitting into monasca dimensions and monasca meta-values
func (sink monascaSink) processLabels(labels map[string]string) (map[string]string, map[string]string) {
	dims := map[string]string{}
	valueMeta := map[string]string{}

	// labels to dimensions
	dims[monascaComponent] = sink.processDimension(labels[sinksApi.LabelPodName.Key])
	dims[monascaHostname] = sink.processDimension(labels[sinksApi.LabelHostname.Key])
	dims[sinksApi.LabelContainerName.Key] = sink.processDimension(labels[sinksApi.LabelContainerName.Key])
	dims[monascaService] = "kubernetes"

	// labels to valueMeta
	for i, v := range labels {
		if i != sinksApi.LabelPodName.Key && i != sinksApi.LabelHostname.Key &&
			i != sinksApi.LabelContainerName.Key && v != "" {
			valueMeta[i] = strings.Replace(v, ",", " ", -1)
		}
	}
	return dims, valueMeta
}

// creates a valid dimension value
func (sink monascaSink) processDimension(value string) string {
	if value != "" {
		v := strings.Replace(value, "/", ".", -1)
		return strings.Replace(v, ",", " ", -1)
	}
	return emptyValue
}

// Monasca does not have API for events.
func (sink monascaSink) StoreEvents([]kube_api.Event) error {
	return nil
}

// Monasca metrics are implicit and do not require registration.
func (sink monascaSink) Register(metrics []sinksApi.MetricDescriptor) error {
	return nil
}

// Monasca metrics are implicit and do not require deregistration
func (sink monascaSink) Unregister(metrics []sinksApi.MetricDescriptor) error {
	return nil
}

func (sink monascaSink) DebugInfo() string {
	return "Sink Type: Monasca"
}

func (sink monascaSink) Name() string {
	return "Monasca Sink"
}

func init() {
	extpoints.SinkFactories.Register(CreateMonascaSink, "monasca")
}

// CreateMonascaSink creates a monasca sink that can consume the Monasca APIs to create metrics.
func CreateMonascaSink(uri *url.URL, _ extpoints.HeapsterConf) ([]sinksApi.ExternalSink, error) {
	opts := uri.Query()
	config := NewConfig(opts)
	client, err := NewMonascaClient(config)
	if err != nil {
		return nil, err
	}
	sink := monascaSink{client: client}
	glog.Infof("Created Monasca sink. Monasca server running on: %s", client.GetURL().String())
	return []sinksApi.ExternalSink{sink}, nil
}
