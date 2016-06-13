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

package push

import (
	"fmt"
	"testing"
	"time"

	. "k8s.io/heapster/metrics/core"

	"github.com/stretchr/testify/assert"
)

func makeCntMetricValue(val int64) MetricValue {
	return MetricValue{
		IntValue:   val,
		ValueType:  ValueInt64,
		MetricType: MetricCumulative,
	}
}

func makeGaugeMetricValue(val float32) MetricValue {
	return MetricValue{
		FloatValue: val,
		ValueType:  ValueFloat,
		MetricType: MetricGauge,
	}
}

func makeBaseBatch1(nowTime time.Time) DataBatch {
	return DataBatch{
		Timestamp: nowTime.Add(-5 * time.Second),
		MetricSets: map[string]*MetricSet{
			PodKey("somens", "somepod"): {
				ScrapeTime: nowTime.Add(-10 * time.Second),
				MetricValues: map[string]MetricValue{
					"custom/adminmetrics/mtn_dew_consumption": makeGaugeMetricValue(12.0),
				},
				Labels: map[string]string{
					LabelNamespaceName.Key: "somens",
					LabelPodName.Key:       "somepod",
				},
				LabeledMetrics: []LabeledMetric{
					{
						Name:        "custom/adminmetrics/doritos_eaten",
						Labels:      map[string]string{"flavor": "original"},
						MetricValue: makeGaugeMetricValue(10.0),
					},
					{
						Name:        "custom/adminmetrics/doritos_eaten",
						Labels:      map[string]string{"flavor": "cool_ranch"},
						MetricValue: makeGaugeMetricValue(10.0),
					},
				},
			},

			NamespaceKey("somens"): {
				ScrapeTime: nowTime.Add(-10 * time.Second),
				MetricValues: map[string]MetricValue{
					"custom/adminmetrics/cans_collected": makeCntMetricValue(48),
				},
				Labels: map[string]string{
					LabelNamespaceName.Key: "somens",
				},
				LabeledMetrics: nil,
			},
		},
	}
}

func makeBaseBatch2(nowTime time.Time) DataBatch {
	return DataBatch{
		Timestamp: nowTime.Add(-5 * time.Second),
		MetricSets: map[string]*MetricSet{
			NamespaceKey("somens"): {
				ScrapeTime: nowTime.Add(-10 * time.Second),
				MetricValues: map[string]MetricValue{
					"custom/routermetrics/frontend_http_hits": makeGaugeMetricValue(20000.0),
					"custom/routermetrics/proxy_http_hits":    makeGaugeMetricValue(500000.0),
				},
				Labels: map[string]string{
					LabelNamespaceName.Key: "somens",
				},
				LabeledMetrics: nil,
			},
			NamespaceKey("somens2"): {
				ScrapeTime: nowTime.Add(-10 * time.Second),
				MetricValues: map[string]MetricValue{
					"custom/routermetrics/frontend_http_hits": makeGaugeMetricValue(20.0),
					"custom/routermetrics/proxy_http_hits":    makeGaugeMetricValue(500.0),
				},
				Labels: map[string]string{
					LabelNamespaceName.Key: "somens2",
				},
				LabeledMetrics: nil,
			},
		},
	}
}

func TestPushMetricsIntoPushSource(t *testing.T) {
	assert := assert.New(t)
	nowTime := time.Now()
	nowFunc = func() time.Time { return nowTime }

	batch1 := makeBaseBatch1(nowTime)
	batchBefore := makeBaseBatch1(nowTime.Add(-2 * time.Minute))
	batchAfter := makeBaseBatch2(nowTime.Add(2 * time.Minute))

	batch1Multi := makeBaseBatch1(nowTime)
	batch1Multi.MetricSets[PodKey("somens", "somepod")].MetricValues["custom/adminmetrics/cheese_sticks_eaten"] = makeGaugeMetricValue(6.5)
	avgedMetric := makeGaugeMetricValue(11.0)
	avgedMetric.IntValue = 2
	batch1Multi.MetricSets[PodKey("somens", "somepod")].MetricValues["custom/adminmetrics/mtn_dew_consumption"] = avgedMetric
	batch1Multi.Timestamp = nowTime

	batch1Multi2 := makeBaseBatch1(nowTime)
	avgedMetric2 := makeGaugeMetricValue(10.0)
	avgedMetric2.IntValue = 2
	avgedMetric2New := makeGaugeMetricValue(32.0 / 3.0)
	avgedMetric2New.IntValue = 3
	batch1Multi2.MetricSets[PodKey("somens", "somepod")].MetricValues["custom/adminmetrics/mtn_dew_consumption"] = avgedMetric2New
	batch1Multi2.Timestamp = nowTime

	batch2 := makeBaseBatch2(nowTime)

	mergedBatch := makeBaseBatch1(nowTime)
	mergedBatch.Timestamp = nowTime
	mergedBatch.MetricSets[NamespaceKey("somens2")] = batch2.MetricSets[NamespaceKey("somens2")]

	for k, v := range batch2.MetricSets[NamespaceKey("somens")].MetricValues {
		mergedBatch.MetricSets[NamespaceKey("somens")].MetricValues[k] = v
	}

	tests := []struct {
		test           string
		inputBatch     *DataBatch
		inputName      string
		presentBatches map[string]DataBatch
		expectedBatch  *DataBatch
		start          time.Time
		end            time.Time
	}{
		{
			test:       "new batch with the same name merges with older batch, overwrites when necessary",
			inputBatch: &batch1,
			inputName:  "adminmetrics",
			presentBatches: map[string]DataBatch{
				"adminmetrics": {
					Timestamp: nowTime.Add(-10 * time.Second),
					MetricSets: map[string]*MetricSet{
						PodKey("somens", "somepod"): {
							ScrapeTime: nowTime.Add(-10 * time.Second),
							MetricValues: map[string]MetricValue{
								"custom/adminmetrics/mtn_dew_consumption": makeGaugeMetricValue(10.0),
								"custom/adminmetrics/cheese_sticks_eaten": makeGaugeMetricValue(6.5),
							},
							Labels: map[string]string{
								LabelNamespaceName.Key: "somens",
								LabelPodName.Key:       "somepod",
							},
						},

						NamespaceKey("somens"): {
							ScrapeTime: nowTime.Add(-10 * time.Second),
							MetricValues: map[string]MetricValue{
								"custom/adminmetrics/cans_collected": makeCntMetricValue(37),
							},
							Labels: map[string]string{
								LabelNamespaceName.Key: "somens",
							},
						},
					},
				},
			},
			expectedBatch: &batch1Multi,
		},
		{
			test:       "new batch with same name merges with older batch (averaging across multiple pushes)",
			inputBatch: &batch1,
			inputName:  "adminmetrics",
			presentBatches: map[string]DataBatch{
				"adminmetrics": {
					Timestamp: nowTime.Add(-10 * time.Second),
					MetricSets: map[string]*MetricSet{
						PodKey("somens", "somepod"): {
							ScrapeTime: nowTime.Add(-10 * time.Second),
							MetricValues: map[string]MetricValue{
								"custom/adminmetrics/mtn_dew_consumption": avgedMetric2,
							},
							Labels: map[string]string{
								LabelNamespaceName.Key: "somens",
								LabelPodName.Key:       "somepod",
							},
						},
					},
				},
			},
			expectedBatch: &batch1Multi2,
		},
		{
			test:       "batch before start is ignored",
			inputBatch: &batchBefore,
			inputName:  "adminmetrics",
			start:      nowTime.Add(-1 * time.Minute),
			expectedBatch: &DataBatch{
				Timestamp:  nowTime,
				MetricSets: make(map[string]*MetricSet),
			},
		},
		{
			test:       "batch after end is ignored",
			inputBatch: &batchAfter,
			inputName:  "adminmetrics",
			end:        nowTime.Add(1 * time.Minute),
			expectedBatch: &DataBatch{
				Timestamp:  nowTime,
				MetricSets: make(map[string]*MetricSet),
			},
		},
		{
			// This test is dependent on the behavior of MergeMetricSet
			test:       "two different sources' batches merge properly",
			inputBatch: &batch2,
			inputName:  "routermetrics",
			presentBatches: map[string]DataBatch{
				"adminmetrics": batch1,
			},
			expectedBatch: &mergedBatch,
		},
	}

	for _, test := range tests {
		if test.presentBatches == nil {
			test.presentBatches = make(map[string]DataBatch)
		}

		src := memoryPushSource{
			dataBatches: test.presentBatches,
		}

		src.PushMetrics(test.inputBatch, test.inputName)
		outputBatch := src.ScrapeMetrics(test.start, test.end)
		if !assert.Equal(test.expectedBatch, outputBatch, fmt.Sprintf("%s: scraped batch was not as expected", test.test)) {
			for k, v := range test.expectedBatch.MetricSets {
				assert.Equal(v, outputBatch.MetricSets[k])
			}
			continue
		}
	}
}
