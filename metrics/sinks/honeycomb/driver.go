// Copyright 2017 Google Inc. All Rights Reserved.
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

package honeycomb

import (
	"net/url"
	"sync"

	"github.com/golang/glog"
	honeycomb_common "k8s.io/heapster/common/honeycomb"
	"k8s.io/heapster/metrics/core"
)

// These metrics report cumulative values over the lifetime of the process.
// Heapster also reports gauges (e.g., "cpu/usage_rate"). Cumulative metrics
// are more confusing than helpful, so let's not send them in the first place.
var blacklist = map[string]struct{}{
	"cpu/usage":                {},
	"memory/major_page_faults": {},
	"memory/page_faults":       {},
	"network/rx_errors":        {},
	"network/rx":               {},
	"network/tx_errors":        {},
	"network/tx":               {},
}

type honeycombSink struct {
	client honeycomb_common.Client
	sync.Mutex
}

type Point struct {
	MetricsName string
	MetricsTags string
}

func (sink *honeycombSink) ExportData(dataBatch *core.DataBatch) {

	sink.Lock()
	defer sink.Unlock()

	batch := make(honeycomb_common.Batch, len(dataBatch.MetricSets))

	i := 0
	for _, metricSet := range dataBatch.MetricSets {
		data := make(map[string]interface{})
		for metricName, metricValue := range metricSet.MetricValues {
			if _, ok := blacklist[metricName]; ok {
				continue
			}
			data[metricName] = metricValue.GetValue()
		}
		for k, v := range metricSet.Labels {
			data[k] = v
		}
		batch[i] = &honeycomb_common.BatchPoint{
			Data:      data,
			Timestamp: dataBatch.Timestamp,
		}
		i++
	}
	err := sink.client.SendBatch(batch)
	if err != nil {
		glog.Warningf("Failed to send metrics batch: %v", err)
	}
}

func (sink *honeycombSink) Stop() {}
func (sink *honeycombSink) Name() string {
	return "Honeycomb Sink"
}

func NewHoneycombSink(uri *url.URL) (core.DataSink, error) {
	client, err := honeycomb_common.NewClient(uri)
	if err != nil {
		return nil, err
	}
	sink := &honeycombSink{
		client: client,
	}

	return sink, nil
}
