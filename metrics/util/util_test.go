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

package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/heapster/metrics/core"
)

func TestMergeMetricSetAlwaysOverwritesZeroTimes(t *testing.T) {
	nowTime := time.Now()
	assert := assert.New(t)

	destSet := &core.MetricSet{
		CreateTime: time.Time{},
		ScrapeTime: time.Time{},
	}

	newSet := &core.MetricSet{
		CreateTime: nowTime.Add(10 * time.Second),
		ScrapeTime: nowTime.Add(20 * time.Second),
	}

	MergeMetricSet(destSet, newSet, false)
	assert.Equal(destSet.CreateTime, newSet.CreateTime, "Zero create time in the destination set should have been overwritten with the non-zero times in the new set")
	assert.Equal(destSet.ScrapeTime, newSet.ScrapeTime, "Zero scrape time in the destination set should have been overwritten with the non-zero times in the new set")
}

func TestMergeMetricSetNoOverwriteWithTimes(t *testing.T) {
	nowTime := time.Now()
	assert := assert.New(t)

	destSet := &core.MetricSet{
		CreateTime: nowTime.Add(10 * time.Second),
		ScrapeTime: nowTime.Add(20 * time.Second),

		Labels: map[string]string{
			"lbl1": "val1",
			"lbl2": "val2",
		},

		MetricValues: map[string]core.MetricValue{
			"metric1": {
				IntValue: 3,
			},
			"metric2": {
				FloatValue: 4.0,
			},
		},

		LabeledMetrics: []core.LabeledMetric{
			{
				Name:   "lblMetric1",
				Labels: map[string]string{"cheese": "cheddar"},
				MetricValue: core.MetricValue{
					IntValue: 3,
				},
			},
		},
	}

	newSet := &core.MetricSet{
		CreateTime: nowTime.Add(30 * time.Second),
		ScrapeTime: nowTime.Add(40 * time.Second),

		Labels: map[string]string{
			"lbl1": "val1.2",
			"lbl3": "val3",
		},

		MetricValues: map[string]core.MetricValue{
			"metric1": {
				IntValue: 5,
			},
			"metric3": {
				FloatValue: 6.0,
			},
		},

		LabeledMetrics: []core.LabeledMetric{
			{
				Name:   "lblMetric1",
				Labels: map[string]string{"cheese": "pepperJack"},
				MetricValue: core.MetricValue{
					IntValue: 4,
				},
			},
		},
	}

	MergeMetricSet(destSet, newSet, false)

	// it should only set times if they're zero
	assert.Equal(destSet.CreateTime, nowTime.Add(10*time.Second), "the destination set's non-zero create time should not have been overwritten")
	assert.Equal(destSet.ScrapeTime, nowTime.Add(20*time.Second), "the destination set's non-zero scrape time should not have been overwritten")

	// it should add new labels and keep existing ones
	expectedLabels := map[string]string{
		"lbl1": "val1",
		"lbl2": "val2",
		"lbl3": "val3",
	}
	assert.Equal(destSet.Labels, expectedLabels, "the destination set's labels should have the new labels from the new set, but should have kept any existing values")

	// it should add new metric values and keep existing ones
	expectedMetricValues := map[string]core.MetricValue{
		"metric1": {
			IntValue: 3,
		},
		"metric2": {
			FloatValue: 4.0,
		},
		"metric3": {
			FloatValue: 6.0,
		},
	}
	assert.Equal(destSet.MetricValues, expectedMetricValues, "the destination set's metric values should have new metric values from the new set, but should have kept any existing values")

	// it should append new labeled metrics
	expectedLabeledMetrics := []core.LabeledMetric{
		{
			Name:   "lblMetric1",
			Labels: map[string]string{"cheese": "cheddar"},
			MetricValue: core.MetricValue{
				IntValue: 3,
			},
		},
		{
			Name:   "lblMetric1",
			Labels: map[string]string{"cheese": "pepperJack"},
			MetricValue: core.MetricValue{
				IntValue: 4,
			},
		},
	}
	assert.Equal(destSet.LabeledMetrics, expectedLabeledMetrics, "the destination set's labeled metrics should have included the labeled metrics it originally had, plus the labeled metrics from the new set")
}

func TestMergeMetricSetOverwrite(t *testing.T) {
	nowTime := time.Now()
	assert := assert.New(t)

	destSet := &core.MetricSet{
		CreateTime: nowTime.Add(10 * time.Second),
		ScrapeTime: nowTime.Add(20 * time.Second),

		Labels: map[string]string{
			"lbl1": "val1",
			"lbl2": "val2",
		},

		MetricValues: map[string]core.MetricValue{
			"metric1": {
				IntValue: 3,
			},
			"metric2": {
				FloatValue: 4.0,
			},
		},

		LabeledMetrics: []core.LabeledMetric{
			{
				Name:   "lblMetric1",
				Labels: map[string]string{"cheese": "cheddar"},
				MetricValue: core.MetricValue{
					IntValue: 3,
				},
			},
		},
	}

	newSet := &core.MetricSet{
		CreateTime: nowTime.Add(30 * time.Second),
		ScrapeTime: nowTime.Add(40 * time.Second),

		Labels: map[string]string{
			"lbl1": "val1.2",
			"lbl3": "val3",
		},

		MetricValues: map[string]core.MetricValue{
			"metric1": {
				IntValue: 5,
			},
			"metric3": {
				FloatValue: 6.0,
			},
		},

		LabeledMetrics: []core.LabeledMetric{
			{
				Name:   "lblMetric1",
				Labels: map[string]string{"cheese": "pepperJack"},
				MetricValue: core.MetricValue{
					IntValue: 4,
				},
			},
		},
	}

	MergeMetricSet(destSet, newSet, true)

	// it should overwrite times regardless
	assert.Equal(destSet.CreateTime, nowTime.Add(30*time.Second), "the destination set's non-zero create time should have been overwritten")
	assert.Equal(destSet.ScrapeTime, nowTime.Add(40*time.Second), "the destination set's non-zero scrape time should have been overwritten")

	// it should add new labels and overwrite exiting ones
	expectedLabels := map[string]string{
		"lbl1": "val1.2",
		"lbl2": "val2",
		"lbl3": "val3",
	}
	assert.Equal(destSet.Labels, expectedLabels, "the destination set's labels should have the new labels from the new set and should have overwritten any existing duplicate label values")

	// it should add new metric values and overwrite existing ones
	expectedMetricValues := map[string]core.MetricValue{
		"metric1": {
			IntValue: 5,
		},
		"metric2": {
			FloatValue: 4.0,
		},
		"metric3": {
			FloatValue: 6.0,
		},
	}
	assert.Equal(destSet.MetricValues, expectedMetricValues, "the destination set's metric values should have new metric values from the new set and should have overwritten any existing duplicate metric values")

	// it should append new labeled metrics
	expectedLabeledMetrics := []core.LabeledMetric{
		{
			Name:   "lblMetric1",
			Labels: map[string]string{"cheese": "cheddar"},
			MetricValue: core.MetricValue{
				IntValue: 3,
			},
		},
		{
			Name:   "lblMetric1",
			Labels: map[string]string{"cheese": "pepperJack"},
			MetricValue: core.MetricValue{
				IntValue: 4,
			},
		},
	}
	assert.Equal(destSet.LabeledMetrics, expectedLabeledMetrics, "the destination set's labeled metrics should have included the labeled metrics it originally had, plus the labeled metrics from the new set")
}
