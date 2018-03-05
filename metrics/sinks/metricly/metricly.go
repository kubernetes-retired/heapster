// Copyright 2018 Google Inc. All Rights Reserved.
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
package metricly

import (
	"github.com/golang/glog"
	"github.com/metricly/go-client/api"
	metricly_core "github.com/metricly/go-client/model/core"
	"k8s.io/heapster/common/metricly"
	"k8s.io/heapster/metrics/core"
	"net/url"
	"strings"
)

type MetriclyMetricsSink struct {
	client api.Client
}

func (sink *MetriclyMetricsSink) Name() string {
	return "Metricly Metrics Sink"
}

func (sink *MetriclyMetricsSink) Stop() {
}

func (sink *MetriclyMetricsSink) ExportData(batch *core.DataBatch) {
	glog.Info("Start exporting data batch to metricly ...")
	elements := DataBatchToElements(batch)
	if err := sink.client.PostElements(elements); err != nil {
		glog.Errorf("Error occurred  during exporting elements with response:  %s", err)
	}
	glog.Info("Exported ", len(elements), " elements")
}

func NewMetriclySink(uri *url.URL) (core.DataSink, error) {
	config, _ := metricly.Config(uri)
	glog.Info("Create metricly sink using config: ", config)
	return &MetriclyMetricsSink{client: api.NewClient(config.ApiURL, config.ApiKey)}, nil
}

func DataBatchToElements(batch *core.DataBatch) []metricly_core.Element {
	ts := batch.Timestamp.Unix() * 1000
	var elements []metricly_core.Element
	for key, ms := range batch.MetricSets {
		etype := ms.Labels["type"]
		element := metricly_core.NewElement(key, key, etype, "")
		// metric set labels to element tags
		for lname, lvalue := range ms.Labels {
			element.AddTag(lname, lvalue)
		}
		// metrics
		for mname, mvalue := range ms.MetricValues {
			if sample, err := metricly_core.NewSample(sanitizeMetricId(mname), ts, mvalue.GetValue()); err == nil {
				element.AddSample(sample)
			}
		}
		// labeled metrics
		for _, lmetric := range ms.LabeledMetrics {
			instanceMetricName := sanitizeMetricId(lmetric.Name) + ":" + lmetric.Labels["resource_id"]
			if sample, err := metricly_core.NewSample(instanceMetricName, ts, lmetric.GetValue()); err == nil {
				element.AddSample(sample)
			}
		}
		elements = append(elements, element)
	}
	LinkElements(elements)
	return elements
}

func LinkElements(elements []metricly_core.Element) {
	var elementsById = make(map[string]*metricly_core.Element)
	for idx := range elements {
		switch e := elements[idx]; e.Type {
		case "pod":
			if id, ok := elements[idx].Tag("pod_id"); ok {
				elementsById[id.Value] = &e
			}
		case "node":
			if id, ok := elements[idx].Tag("host_id"); ok {
				elementsById[id.Value] = &e
			}
		case "ns":
			if id, ok := elements[idx].Tag("namespace_id"); ok {
				elementsById[id.Value] = &e
			}
		}
	}

	for idx := range elements {
		switch e := elements[idx]; e.Type {
		case "pod_container":
			if podId, ok := e.Tag("pod_id"); ok {
				if pod, ok := elementsById[podId.Value]; ok {
					pod.AddRelation(e.Id)
				}
			}
			if hostId, ok := elements[idx].Tag("host_id"); ok {
				if host, ok := elementsById[hostId.Value]; ok {
					host.AddRelation(e.Id)
				}
			}
			if nsId, ok := elements[idx].Tag("namespace_id"); ok {
				if ns, ok := elementsById[nsId.Value]; ok {
					ns.AddRelation(e.Id)
				}
			}
		case "pod":
			if hostId, ok := elements[idx].Tag("host_id"); ok {
				if host, ok := elementsById[hostId.Value]; ok {
					host.AddRelation(e.Id)
				}
			}
			if nsId, ok := elements[idx].Tag("namespace_id"); ok {
				if ns, ok := elementsById[nsId.Value]; ok {
					ns.AddRelation(e.Id)
				}
			}
		case "sys_container":
			if hostId, ok := elements[idx].Tag("host_id"); ok {
				if host, ok := elementsById[hostId.Value]; ok {
					host.AddRelation(e.Id)
				}
			}
		}
	}
}

func sanitizeMetricId(metricId string) string {
	return strings.Replace(metricId, "/", ".", -1)
}
