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
	"testing"

	"k8s.io/heapster/metrics/core"

	"github.com/stretchr/testify/assert"
)

// MergeInto and DataBatch are exercised by the main push tests

func TestMetricValueFinalValue(t *testing.T) {
	assert := assert.New(t)

	// non-gauge metrics
	inputCumulative := &MetricValue{
		MetricValue: core.MetricValue{
			MetricType: core.MetricCumulative,
			FloatValue: 10.0,
			IntValue:   5,
		},
	}
	assert.Equal(inputCumulative.MetricValue, inputCumulative.FinalValue(), "non-gauge metrics should return the value without modification")

	// guage metrics with count of zero
	inputGaugeSingle := &MetricValue{
		MetricValue: core.MetricValue{
			ValueType:  core.ValueFloat,
			MetricType: core.MetricGauge,
			FloatValue: 10,
		},
	}
	assert.Equal(inputGaugeSingle.MetricValue, inputGaugeSingle.FinalValue(), "gauge metrics with a count of less than 2 should return the value without modification")

	// gauge float values
	inputGaugeFloat := &MetricValue{
		StoredCount: 3,
		MetricValue: core.MetricValue{
			ValueType:  core.ValueFloat,
			MetricType: core.MetricGauge,
			FloatValue: 30,
		},
	}
	expectedGaugeFloat := core.MetricValue{
		ValueType:  core.ValueFloat,
		MetricType: core.MetricGauge,
		FloatValue: 10,
	}
	assert.Equal(expectedGaugeFloat, inputGaugeFloat.FinalValue(), "float guage metrics should return the average value")

	// gauge int values
	inputGaugeInt := &MetricValue{
		StoredCount: 3,
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricGauge,
			IntValue:   30,
		},
	}
	expectedGaugeInt := core.MetricValue{
		ValueType:  core.ValueInt64,
		MetricType: core.MetricGauge,
		IntValue:   10,
	}
	assert.Equal(expectedGaugeInt, inputGaugeInt.FinalValue(), "int guage metrics should return the average value")
}

func TestMetricValueAppend(t *testing.T) {
	assert := assert.New(t)

	// base values
	origValNonGauge := MetricValue{
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricCumulative,
			IntValue:   10,
		},
	}
	origValZero := MetricValue{
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricGauge,
			IntValue:   10,
		},
	}
	origValMultInt := MetricValue{
		StoredCount: 2,
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricGauge,
			IntValue:   20,
		},
	}
	origValMultFloat := MetricValue{
		StoredCount: 2,
		MetricValue: core.MetricValue{
			ValueType:  core.ValueFloat,
			MetricType: core.MetricGauge,
			FloatValue: 20,
		},
	}

	// new values
	newValFloat := MetricValue{
		MetricValue: core.MetricValue{
			ValueType:  core.ValueFloat,
			MetricType: core.MetricGauge,
			FloatValue: 10,
		},
	}
	newValInt := MetricValue{
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricGauge,
			IntValue:   10,
		},
	}

	// expected vals
	expectedValMultInt := MetricValue{
		StoredCount: 3,
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricGauge,
			IntValue:   30,
		},
	}
	expectedValMultFloat := MetricValue{
		StoredCount: 3,
		MetricValue: core.MetricValue{
			ValueType:  core.ValueFloat,
			MetricType: core.MetricGauge,
			FloatValue: 30,
		},
	}
	expectedValTwo := MetricValue{
		StoredCount: 2,
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricGauge,
			IntValue:   20,
		},
	}

	assert.Equal(newValFloat, origValNonGauge.Append(newValFloat), "non-gauge values should simply return the new value")
	assert.Equal(expectedValTwo, origValZero.Append(newValInt), "gauge metrics with a count of zero should jump to a count of two")
	assert.Equal(expectedValMultFloat, origValMultFloat.Append(newValFloat), "float gauge metrics should increment the count and add the float values")
	assert.Equal(expectedValMultInt, origValMultInt.Append(newValInt), "int gauge metrics should increment the count and add the int values")
}

func TestLabeledMetricWrappers(t *testing.T) {
	initialLabeledMetric := LabeledMetric{
		Name:   "cheese",
		Labels: []LabelPair{{"color", "white"}, {"sharpness", "extra"}},
		MetricValue: MetricValue{
			StoredCount: 2,
			MetricValue: core.MetricValue{
				ValueType:  core.ValueInt64,
				MetricType: core.MetricGauge,
				IntValue:   20,
			},
		},
	}
	newLabeledMetric := LabeledMetric{
		Name:   "cheese",
		Labels: []LabelPair{{"color", "white"}, {"sharpness", "extra"}},
		MetricValue: MetricValue{
			MetricValue: core.MetricValue{
				ValueType:  core.ValueInt64,
				MetricType: core.MetricGauge,
				IntValue:   10,
			},
		},
	}
	expectedLabeledMetric := core.LabeledMetric{
		Name:   "cheese",
		Labels: map[string]string{"sharpness": "extra", "color": "white"},
		MetricValue: core.MetricValue{
			ValueType:  core.ValueInt64,
			MetricType: core.MetricGauge,
			IntValue:   10,
		},
	}

	assert.Equal(t, expectedLabeledMetric, initialLabeledMetric.Append(newLabeledMetric).FinalValue(), "calling the same methods on labeled metrics as on normal metric values should work like calling the methods on their values and preserving the rest")
}

func TestLabeledMetricMakeID(t *testing.T) {
	metric := LabeledMetric{
		Name:   "cheese",
		Labels: []LabelPair{{"color", "white"}, {"sharpness", "extra"}},
	}

	expectedID := LabeledMetricID{
		Name:         "cheese",
		JoinedLabels: "color\000white\000sharpness\000extra",
	}

	assert.Equal(t, expectedID, metric.MakeID(), "the ID of the labeled metric should consist of its name, plus its labels joined together")
}
