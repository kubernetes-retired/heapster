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
	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/metrics/core"
	"testing"
)

var (
	influxPrefix       = "testprefix."
	influxMetricName   = "testmetric"
	influxResourceName = "testresource"
)

var influxLabels = map[string]string{
	"test_tag_1": "value1",
	"test_tag_2": "value2",
	"test_tag_3": "value3",
}

var influxMetricValue = core.MetricValue{
	MetricType: core.MetricGauge,
	ValueType:  core.ValueInt64,
	IntValue:   1000,
}

func TestInfluxFormatWithoutLabels(t *testing.T) {
	expectedMsg := "testprefix.testmetric:1000|g"

	formatter := NewInfluxstatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(influxPrefix, influxMetricName, nil, SnakeToLowerCamel, influxMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestInfluxFormatWithLabels(t *testing.T) {
	expectedMsg := "testprefix.testmetric,testTag1=value1,testTag2=value2,testTag3=value3:1000|g"

	formatter := NewInfluxstatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(influxPrefix, influxMetricName, influxLabels, SnakeToLowerCamel, influxMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestInfluxFormatWithoutPrefix(t *testing.T) {
	expectedMsg := "testmetric,TestTag1=value1,TestTag2=value2,TestTag3=value3:1000|g"

	formatter := NewInfluxstatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format("", influxMetricName, influxLabels, SnakeToUpperCamel, influxMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}
