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

	"k8s.io/heapster/metrics/core"

	"github.com/stretchr/testify/assert"
)

func makeCntMetricValue(val int64) MetricValue {
	return MetricValue{
		MetricValue: core.MetricValue{
			IntValue:   val,
			ValueType:  core.ValueInt64,
			MetricType: core.MetricCumulative,
		},
	}
}

func makeGaugeMetricValue(val float32) MetricValue {
	return MetricValue{
		MetricValue: core.MetricValue{
			FloatValue: val,
			ValueType:  core.ValueFloat,
			MetricType: core.MetricGauge,
		},
	}
}

func makeLabeledMetrics(metrics []LabeledMetric) map[LabeledMetricID]LabeledMetric {
	res := make(map[LabeledMetricID]LabeledMetric, len(metrics))
	for _, metric := range metrics {
		res[metric.MakeID()] = metric
	}

	return res
}

func makeBaseBatch1(nowTime time.Time) (DataBatch, map[string]struct{}) {
	return DataBatch{
			Timestamp: nowTime.Add(-5 * time.Second),
			MetricSets: map[string]*MetricSet{
				core.PodKey("somens", "somepod"): {
					MetricValues: map[string]MetricValue{
						"custom/adminmetrics/mtn_dew_consumption": makeGaugeMetricValue(12.0),
					},
					Labels: map[string]string{
						core.LabelNamespaceName.Key: "somens",
						core.LabelPodName.Key:       "somepod",
					},
					LabeledMetrics: makeLabeledMetrics([]LabeledMetric{
						{
							Name:        "custom/adminmetrics/doritos_eaten",
							Labels:      []LabelPair{{"flavor", "original"}},
							MetricValue: makeGaugeMetricValue(10.0),
						},
						{
							Name:        "custom/adminmetrics/doritos_eaten",
							Labels:      []LabelPair{{"flavor", "cool_ranch"}},
							MetricValue: makeGaugeMetricValue(10.0),
						},
					}),
				},

				core.NamespaceKey("somens"): {
					MetricValues: map[string]MetricValue{
						"custom/adminmetrics/cans_collected": makeCntMetricValue(48),
					},
					Labels: map[string]string{
						core.LabelNamespaceName.Key: "somens",
					},
					LabeledMetrics: nil,
				},
			},
		}, map[string]struct{}{
			"custom/adminmetrics/mtn_dew_consumption": {},
			"custom/adminmetrics/doritos_eaten":       {},
			"custom/adminmetrics/cans_collected":      {},
		}
}

func makeBaseBatch2(nowTime time.Time) (DataBatch, map[string]struct{}) {
	return DataBatch{
			Timestamp: nowTime.Add(-5 * time.Second),
			MetricSets: map[string]*MetricSet{
				core.NamespaceKey("somens"): {
					MetricValues: map[string]MetricValue{
						"custom/routermetrics/frontend_http_hits": makeGaugeMetricValue(20000.0),
						"custom/routermetrics/proxy_http_hits":    makeGaugeMetricValue(500000.0),
					},
					Labels: map[string]string{
						core.LabelNamespaceName.Key: "somens",
					},
					LabeledMetrics: nil,
				},
				core.NamespaceKey("somens2"): {
					MetricValues: map[string]MetricValue{
						"custom/routermetrics/frontend_http_hits": makeGaugeMetricValue(20.0),
						"custom/routermetrics/proxy_http_hits":    makeGaugeMetricValue(500.0),
					},
					Labels: map[string]string{
						core.LabelNamespaceName.Key: "somens2",
					},
					LabeledMetrics: nil,
				},
			},
		}, map[string]struct{}{
			"custom/routermetrics/frontend_http_hits": {},
			"custom/routermetrics/proxy_http_hits":    {},
		}
}

func TestPushMetricsOverLimit(t *testing.T) {
	assert := assert.New(t)
	src := memoryPushSource{
		dataBatches:  nil,
		metricsLimit: 3,
		metricNames: map[string]map[string]struct{}{
			"somesource": {
				"m1": {},
				"m2": {},
			},
		},
	}

	// batch is irrelevant here
	err := src.PushMetrics(nil, "somesource", map[string]struct{}{"m3": {}, "m4": {}})

	assert.Error(err, "should have thrown an error indicating the metrics limit was reach")
	outputBatch := src.ScrapeMetrics(time.Time{}, time.Time{})

	assert.Empty(outputBatch.MetricSets)

}

func TestPushMetricsIntoPushSource(t *testing.T) {
	nowTime := time.Now()
	nowFunc = func() time.Time { return nowTime }

	batch1, batch1Names := makeBaseBatch1(nowTime)
	batchBefore, batchBeforeNames := makeBaseBatch1(nowTime.Add(-2 * time.Minute))
	batchAfter, batchAfterNames := makeBaseBatch2(nowTime.Add(2 * time.Minute))

	batch1Multi, _ := makeBaseBatch1(nowTime)
	batch1Multi.MetricSets[core.PodKey("somens", "somepod")].MetricValues["custom/adminmetrics/cheese_sticks_eaten"] = makeGaugeMetricValue(6.5)
	avgedMetric := makeGaugeMetricValue(11.0)
	batch1Multi.MetricSets[core.PodKey("somens", "somepod")].MetricValues["custom/adminmetrics/mtn_dew_consumption"] = avgedMetric
	batch1Multi.Timestamp = nowTime

	batch1Multi2, _ := makeBaseBatch1(nowTime)
	avgedMetric2 := makeGaugeMetricValue(20.0)
	avgedMetric2.StoredCount = 2
	avgedMetric2New := makeGaugeMetricValue(32.0 / 3.0)
	batch1Multi2.MetricSets[core.PodKey("somens", "somepod")].MetricValues["custom/adminmetrics/mtn_dew_consumption"] = avgedMetric2New
	batch1Multi2.Timestamp = nowTime

	batch2, batch2Names := makeBaseBatch2(nowTime)

	mergedBatch, _ := makeBaseBatch1(nowTime)
	mergedBatch.Timestamp = nowTime
	mergedBatch.MetricSets[core.NamespaceKey("somens2")] = batch2.MetricSets[core.NamespaceKey("somens2")]

	for k, v := range batch2.MetricSets[core.NamespaceKey("somens")].MetricValues {
		mergedBatch.MetricSets[core.NamespaceKey("somens")].MetricValues[k] = v
	}

	tests := []struct {
		test             string
		inputBatch       *DataBatch
		inputMetricNames map[string]struct{}
		inputName        string
		presentBatches   map[string]DataBatch
		expectedBatch    *core.DataBatch
		start            time.Time
		end              time.Time
	}{
		{
			test:             "new batch with the same name merges with older batch, overwrites when necessary",
			inputBatch:       &batch1,
			inputName:        "adminmetrics",
			inputMetricNames: batch1Names,
			presentBatches: map[string]DataBatch{
				"adminmetrics": {
					Timestamp: nowTime.Add(-10 * time.Second),
					MetricSets: map[string]*MetricSet{
						core.PodKey("somens", "somepod"): {
							MetricValues: map[string]MetricValue{
								"custom/adminmetrics/mtn_dew_consumption": makeGaugeMetricValue(10.0),
								"custom/adminmetrics/cheese_sticks_eaten": makeGaugeMetricValue(6.5),
							},
							Labels: map[string]string{
								core.LabelNamespaceName.Key: "somens",
								core.LabelPodName.Key:       "somepod",
							},
						},

						core.NamespaceKey("somens"): {
							MetricValues: map[string]MetricValue{
								"custom/adminmetrics/cans_collected": makeCntMetricValue(37),
							},
							Labels: map[string]string{
								core.LabelNamespaceName.Key: "somens",
							},
						},
					},
				},
			},
			expectedBatch: batch1Multi.DataBatch(),
		},
		{
			test:       "new batch with same name merges with older batch (averaging across multiple pushes)",
			inputBatch: &batch1,
			inputName:  "adminmetrics",
			presentBatches: map[string]DataBatch{
				"adminmetrics": {
					Timestamp: nowTime.Add(-10 * time.Second),
					MetricSets: map[string]*MetricSet{
						core.PodKey("somens", "somepod"): {
							MetricValues: map[string]MetricValue{
								"custom/adminmetrics/mtn_dew_consumption": avgedMetric2,
							},
							Labels: map[string]string{
								core.LabelNamespaceName.Key: "somens",
								core.LabelPodName.Key:       "somepod",
							},
						},
					},
				},
			},
			expectedBatch: batch1Multi2.DataBatch(),
		},
		{
			test:             "batch before start is ignored",
			inputBatch:       &batchBefore,
			inputMetricNames: batchBeforeNames,
			inputName:        "adminmetrics",
			start:            nowTime.Add(-1 * time.Minute),
			expectedBatch: &core.DataBatch{
				Timestamp:  nowTime,
				MetricSets: make(map[string]*core.MetricSet),
			},
		},
		{
			test:             "batch after end is ignored",
			inputBatch:       &batchAfter,
			inputMetricNames: batchAfterNames,
			inputName:        "adminmetrics",
			end:              nowTime.Add(1 * time.Minute),
			expectedBatch: &core.DataBatch{
				Timestamp:  nowTime,
				MetricSets: make(map[string]*core.MetricSet),
			},
		},
		{
			test:             "two different sources' batches merge properly",
			inputBatch:       &batch2,
			inputMetricNames: batch2Names,
			inputName:        "routermetrics",
			presentBatches: map[string]DataBatch{
				"adminmetrics": batch1,
			},
			expectedBatch: mergedBatch.DataBatch(),
		},
	}

	for _, test := range tests {
		if test.presentBatches == nil {
			test.presentBatches = make(map[string]DataBatch)
		}

		src := memoryPushSource{
			dataBatches: test.presentBatches,
		}

		src.PushMetrics(test.inputBatch, test.inputName, test.inputMetricNames)
		outputBatch := src.ScrapeMetrics(test.start, test.end)

		// there's no assigned iteration order for maps, so labeled-metric-map --> labeled-metric-slice can
		// yield different results each time, so we need a custom helper here
		if !assertBatchesEqual(t, test.expectedBatch, outputBatch, "%s: scraped batch was not as expected", test.test) {
			continue
		}
	}
}

func metricSetEqual(expected, actual *core.MetricSet) bool {
	if !expected.CreateTime.Equal(actual.CreateTime) {
		return false
	}

	if !expected.ScrapeTime.Equal(actual.ScrapeTime) {
		return false
	}

	if !assert.ObjectsAreEqual(expected.MetricValues, actual.MetricValues) {
		return false
	}

	if !assert.ObjectsAreEqual(expected.Labels, actual.Labels) {
		return false
	}

	// there's no assigned iteration order for maps, so labeled-metric-map --> labeled-metric-slice can
	// yield different results each time, so just check that the contents are equal

	if len(expected.LabeledMetrics) != len(actual.LabeledMetrics) {
		return false
	}

	for _, expectedMetric := range expected.LabeledMetrics {
		found := false

		for _, actualMetric := range actual.LabeledMetrics {
			if assert.ObjectsAreEqual(expectedMetric, actualMetric) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func assertBatchesEqual(t *testing.T, expectedBatch, actualBatch *core.DataBatch, msgAndArgs ...interface{}) bool {
	if !actualBatch.Timestamp.Equal(expectedBatch.Timestamp) {
		assert.Fail(t, fmt.Sprintf("Not equal: %#v (expected)\n        != %#v (actual)", expectedBatch, actualBatch), msgAndArgs...)
		return false
	}

	if expectedBatch.MetricSets == nil && actualBatch.MetricSets == nil {
		return true
	}

	if len(expectedBatch.MetricSets) != len(actualBatch.MetricSets) {
		assert.Fail(t, fmt.Sprintf("Not equal: %#v (expected)\n        != %#v (actual)", expectedBatch, actualBatch), msgAndArgs...)
		return false
	}

	for setKey, expectedMetricSet := range expectedBatch.MetricSets {
		actualMetricSet, setPresent := actualBatch.MetricSets[setKey]
		if !setPresent {
			assert.Fail(t, fmt.Sprintf("Not equal: %+v (expected)\n        != %+v (actual)", expectedBatch, actualBatch), msgAndArgs...)
			return false
		}

		if expectedMetricSet == nil && actualMetricSet == nil {
			continue
		} else if expectedMetricSet == nil || actualMetricSet == nil {
			assert.Fail(t, fmt.Sprintf("Not equal: %+v (expected)\n        != %+v (actual)", expectedBatch, actualBatch), msgAndArgs...)
			return false
		}

		if !metricSetEqual(expectedMetricSet, actualMetricSet) {
			assert.Fail(t, fmt.Sprintf("Not equal: %+v (expected)\n        != %+v (actual)", expectedBatch, actualBatch), msgAndArgs...)
			return false
		}
	}

	return true
}
