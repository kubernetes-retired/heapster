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

package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMaxTimestamp(t *testing.T) {
	assert := assert.New(t)
	past := time.Unix(1434212566, 0)
	future := time.Unix(1434212800, 0)
	assert.Equal(maxTimestamp(past, future), future)
	assert.Equal(maxTimestamp(future, past), future)
	assert.Equal(maxTimestamp(future, future), future)
}

func TestAddMetricToMapExistingKey(t *testing.T) {
	/* Tests the flow where the metric_name is already present in the map */
	var (
		metrics                   = make(map[string][]*MetricTimeseries)
		new_metric_name string    = "name_already_in_map"
		stamp           time.Time = time.Now()
		value           uint64    = 1234567890
	)

	old_metric := MetricTimeseries{time.Now(), value + 1}
	metrics[new_metric_name] = []*MetricTimeseries{&old_metric}

	assert := assert.New(t)
	assert.NotEmpty(metrics)
	assert.NoError(addMetricToMap(new_metric_name, stamp, value, &metrics))

	assert.Equal(metrics[new_metric_name][0], &old_metric)
	assert.Equal(metrics[new_metric_name][1].Timestamp, stamp)
	assert.Equal(metrics[new_metric_name][1].Value, value)
	assert.Equal(len(metrics[new_metric_name]), 2)
}

func TestAddMetricToMapNewKey(t *testing.T) {
	/* Tests the flow where the metric_name is not present in the map */
	var (
		metrics                   = make(map[string][]*MetricTimeseries)
		new_metric_name string    = "name_not_in_map"
		stamp           time.Time = time.Now()
		value           uint64    = 1234567890
	)
	assert := assert.New(t)
	assert.Empty(metrics)
	assert.NoError(addMetricToMap(new_metric_name, stamp, value, &metrics))

	expected_slice := []*MetricTimeseries{&MetricTimeseries{stamp, value}}
	assert.Equal(metrics[new_metric_name], expected_slice)
}

func TestNewInfoType(t *testing.T) {
	/* Tests the flow where the metric_name is not present in the map */
	var (
		metrics = make(map[string][]*MetricTimeseries)
		labels  = make(map[string]string)
	)
	new_metric := &MetricTimeseries{time.Unix(124124124, 0), uint64(5555)}
	metrics["test"] = []*MetricTimeseries{new_metric}
	labels["name"] = "test"
	assert := assert.New(t)

	new_infotype := newInfoType(nil, nil) // Invoke with no parameters
	assert.Empty(new_infotype.Metrics)
	assert.Empty(new_infotype.Labels)
	assert.NotNil(new_infotype.lock)

	new_infotype = newInfoType(metrics, labels) // Invoke with both parameters
	assert.Equal(new_infotype.Metrics, metrics)
	assert.Equal(new_infotype.Labels, labels)
	assert.NotNil(new_infotype.lock)
}
