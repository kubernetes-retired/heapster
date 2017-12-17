// Copyright 2016 Google Inc. All Rights Reserved.
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

package statsd

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/metrics/core"
)

const (
	driverUrl = "udp://127.0.0.1:4125?protocolType=influxstatsd&allowedLabels=tag1,tag3"
)

func TestDriverName(t *testing.T) {
	url, err := url.Parse(driverUrl)
	assert.NoError(t, err)

	sink, err := NewStatsdSink(url)
	assert.NoError(t, err)
	assert.NotNil(t, sink)

	assert.Equal(t, "StatsD Sink", sink.Name())
}

func TestDriverExportData(t *testing.T) {
	url, err := url.Parse(driverUrl)
	assert.NoError(t, err)

	client := &dummyStatsdClientImpl{messages: nil}
	sink, err := NewStatsdSinkWithClient(url, client)
	assert.NoError(t, err)
	assert.NotNil(t, sink)

	timestamp := time.Now()

	m1 := "test.metric.1"
	m2 := "test.metric.2"
	m3 := "test.metric.3"
	m4 := "test.metric.4"

	var labels = map[string]string{
		"tag1": "value1",
		"tag2": "value2",
		"tag3": "value3",
	}

	labelStr := "tag1=value1,tag3=value3"
	expectedMsgs := [...]string{
		fmt.Sprintf("%s,%s:1|g\n", m1, labelStr),
		fmt.Sprintf("%s,%s:2|g\n", m2, labelStr),
		fmt.Sprintf("%s,%s:3|g\n", m3, labelStr),
		fmt.Sprintf("%s,%s:4|g\n", m4, labelStr),
	}
	metricSet1 := core.MetricSet{
		Labels: labels,
		MetricValues: map[string]core.MetricValue{
			m1: {
				ValueType:  core.ValueInt64,
				MetricType: core.MetricGauge,
				IntValue:   1,
			},
			m2: {
				ValueType:  core.ValueInt64,
				MetricType: core.MetricGauge,
				IntValue:   2,
			},
		},
	}

	metricSet2 := core.MetricSet{
		Labels: labels,
		MetricValues: map[string]core.MetricValue{
			m3: {
				ValueType:  core.ValueInt64,
				MetricType: core.MetricGauge,
				IntValue:   3,
			},
			m4: {
				ValueType:  core.ValueInt64,
				MetricType: core.MetricGauge,
				IntValue:   4,
			},
		},
	}

	dataBatch := &core.DataBatch{
		Timestamp: timestamp,
		MetricSets: map[string]*core.MetricSet{
			"pod1": &metricSet1,
			"pod2": &metricSet2,
		},
	}

	sink.ExportData(dataBatch)

	res := strings.Join(client.messages, "\n") + "\n"
	for _, expectedMsg := range expectedMsgs[:] {
		assert.Contains(t, res, expectedMsg)
	}
}
