// Copyright 2014 Google Inc. All Rights Reserved.
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

package riemann

import (
	"net/url"
	"sync"

	riemann_api "github.com/bigdatadev/goryman"
	"github.com/golang/glog"
	riemannCommon "k8s.io/heapster/common/riemann"
	"k8s.io/heapster/metrics/core"
)

// Abstracted for testing: this package works against any client that obeys the
// interface contract exposed by the goryman Riemann client

type RiemannSink struct {
	client riemannCommon.RiemannClient
	config riemannCommon.RiemannConfig
	sync.RWMutex
}

func CreateRiemannSink(uri *url.URL) (core.DataSink, error) {
	var sink, err = riemannCommon.CreateRiemannSink(uri)
	if err != nil {
		return nil, err
	}
	rs := &RiemannSink{
		client: sink.Client,
		config: sink.Config,
	}
	return rs, nil
}

// Return a user-friendly string describing the sink
func (sink *RiemannSink) Name() string {
	return "Riemann Sink"
}

func (sink *RiemannSink) Stop() {
	sink.client.Close()
}

// ExportData Send a collection of Timeseries to Riemann
func (sink *RiemannSink) ExportData(dataBatch *core.DataBatch) {
	sink.Lock()
	defer sink.Unlock()

	if sink.client == nil {
		var client, err = riemannCommon.SetupRiemannClient(sink.config)
		if err != nil {
			glog.Warningf("Riemann sink not connected: %v", err)
			return
		} else {
			sink.client = client
		}
	}

	var dataEvents []riemann_api.Event
	appendMetric := func(host, name string, value interface{}, labels map[string]string) {
		event := riemann_api.Event{
			Time:        dataBatch.Timestamp.Unix(),
			Service:     name,
			Host:        host,
			Description: "", //no description - waste of bandwidth.
			Attributes:  labels,
			Metric:      value,
			Ttl:         sink.config.Ttl,
			State:       sink.config.State,
			Tags:        sink.config.Tags,
		}

		dataEvents = append(dataEvents, event)
		if len(dataEvents) >= riemannCommon.MaxSendBatchSize {
			riemannCommon.SendData(sink.client, dataEvents)
			dataEvents = nil
		}
	}

	for _, metricSet := range dataBatch.MetricSets {
		host := metricSet.Labels[core.LabelHostname.Key]
		for metricName, metricValue := range metricSet.MetricValues {
			if value := metricValue.GetValue(); value != nil {
				appendMetric(host, metricName, riemannCommon.RiemannValue(value), metricSet.Labels)
			}
		}
		for _, metric := range metricSet.LabeledMetrics {
			if value := metric.GetValue(); value != nil {
				labels := make(map[string]string)
				for k, v := range metricSet.Labels {
					labels[k] = v
				}
				for k, v := range metric.Labels {
					labels[k] = v
				}
				appendMetric(host, metric.Name, riemannCommon.RiemannValue(value), labels)
			}
		}
	}

	if len(dataEvents) > 0 {
		riemannCommon.SendData(sink.client, dataEvents)
	}
}
