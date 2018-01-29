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

package plugin

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	plugin_common "k8s.io/heapster/common/plugin"
	"k8s.io/heapster/metrics/core"
)

var (
	errExpected = errors.New("know Error")
)

type testMetricSink struct{}

func (sink *testMetricSink) WriteMetricsPoint(point *PluginSinkPoint) error {
	return errExpected
}

type fakeMetricSink struct {
	batchPoints []*PluginSinkPoint
}

func (sink *fakeMetricSink) WriteMetricsPoint(point *PluginSinkPoint) error {
	sink.batchPoints = append(sink.batchPoints, point)
	return nil
}

func TestGoPluginSinkWritePoint(t *testing.T) {
	uri, err := url.Parse(fmt.Sprintf("plugin:?cmd=%s&cmd=-test.run=TestHelperProcess&cmd=--", os.Args[0]))
	assert.NoError(t, err)
	sink, err := NewGoPluginSink(uri)
	assert.NoError(t, err)
	defer sink.Stop()

	err = sink.(*GoPluginSink).WriteMetricsPoint(&PluginSinkPoint{})
	assert.True(t, strings.Contains(err.Error(), errExpected.Error()))
}

func TestExportDataEmpty(t *testing.T) {
	fakeSink := &fakeMetricSink{}
	sink := &GoPluginSink{
		MetricSink: fakeSink,
	}
	dataBatch := core.DataBatch{}
	sink.ExportData(&dataBatch)
	assert.Equal(t, 0, len(fakeSink.batchPoints))
}

func TestExportDataContent(t *testing.T) {
	fakeSink := &fakeMetricSink{}
	sink := &GoPluginSink{
		MetricSink: fakeSink,
	}
	timestamp := time.Now()
	var testCases = []struct {
		input    core.DataBatch
		expected []PluginSinkPoint
	}{
		{
			core.DataBatch{
				Timestamp: timestamp,
				MetricSets: map[string]*core.MetricSet{
					"pod1": &core.MetricSet{
						Labels: map[string]string{
							"namespace_id": "123",
						},
						MetricValues: map[string]core.MetricValue{
							"test/metric/1": {
								ValueType:  core.ValueInt64,
								MetricType: core.MetricCumulative,
								IntValue:   123456,
							},
						},
						LabeledMetrics: []core.LabeledMetric{
							{
								Name: "test/metric/2",
								Labels: map[string]string{
									"namespace_id": "345",
								},
								MetricValue: core.MetricValue{
									ValueType:  core.ValueFloat,
									MetricType: core.MetricCumulative,
									FloatValue: 1.02,
								},
							},
						},
					},
				},
			},
			[]PluginSinkPoint{
				PluginSinkPoint{
					MetricsName: "test/metric/1",
					MetricsValue: core.MetricValue{
						IntValue:  123456,
						ValueType: core.ValueInt64,
					},
					MetricsTimestamp: timestamp.UTC(),
					MetricsTags: map[string]string{
						"namespace_id": "123",
					},
				},
				PluginSinkPoint{
					MetricsName: "test/metric/2",
					MetricsValue: core.MetricValue{
						FloatValue: 1.02,
						ValueType:  core.ValueFloat,
					},
					MetricsTimestamp: timestamp.UTC(),
					MetricsTags: map[string]string{
						"namespace_id": "345",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		sink.ExportData(&testCase.input)
		for idx, batchPoint := range fakeSink.batchPoints {
			assert.Equal(t, &testCase.expected[idx], batchPoint)
		}
		fakeSink.batchPoints = fakeSink.batchPoints[:0]
	}
}

func TestExportData(t *testing.T) {
	fakeSink := &fakeMetricSink{}
	sink := &GoPluginSink{
		MetricSink: fakeSink,
	}
	timestamp := time.Now()

	data := core.DataBatch{
		Timestamp: timestamp,
		MetricSets: map[string]*core.MetricSet{
			"pod1": &core.MetricSet{
				Labels: map[string]string{
					"namespace_id": "123",
				},
				MetricValues: map[string]core.MetricValue{
					"test/metric/1": {
						ValueType:  core.ValueInt64,
						MetricType: core.MetricCumulative,
						IntValue:   123456,
					},
				},
			},
			"pod2": &core.MetricSet{
				Labels: map[string]string{
					"namespace_id": "123",
				},
				MetricValues: map[string]core.MetricValue{
					"test/metric/1": {
						ValueType:  core.ValueInt64,
						MetricType: core.MetricCumulative,
						IntValue:   123456,
					},
				},
			},
		},
	}

	sink.ExportData(&data)
	assert.Equal(t, 2, len(fakeSink.batchPoints))
}

// Fake test to be used as a plugin
func TestHelperProcess(t *testing.T) {
	if os.Getenv(plugin_common.Handshake.MagicCookieKey) == "" {
		return
	}

	defer os.Exit(0)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin_common.Handshake,
		Plugins: map[string]plugin.Plugin{
			"metrics": &MetricSinkPlugin{Impl: &testMetricSink{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
