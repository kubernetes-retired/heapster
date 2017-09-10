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
	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/metrics/core"
	"testing"
)

var (
	etsyPrefix               = "testprefix"
	etsyMetricName           = "testmetric"
	etsyHostName             = "testhost"
	etsyNamespace            = "testnamespace"
	etsyPodName              = "testpodname"
	etsyContainerName        = "testcontainername"
	etsyResourceName         = "testresource"
	etsyUserLabels           = "test_tag_1:val1,test_tag_2:val2,test_tag_3:val3"
	expectedLowerCamelLabels = "testTag1.val1.testTag2.val2.testTag3.val3"
	expectedUpperCamelLabels = "TestTag1.val1.TestTag2.val2.TestTag3.val3"
)

var etsyLabels = map[string]string{
	core.LabelHostname.Key:      etsyHostName,
	core.LabelNamespaceName.Key: etsyNamespace,
	core.LabelPodName.Key:       etsyPodName,
	core.LabelContainerName.Key: etsyContainerName,
	core.LabelLabels.Key:        etsyUserLabels,
}

var etsyMetricValue = core.MetricValue{
	MetricType: core.MetricGauge,
	ValueType:  core.ValueInt64,
	IntValue:   1000,
}

var renameLabels = map[string]string{"old1": "new1", "old2": "new_label_2", "old3": "new_label_3"}
var defaultCustomizer = LabelCustomizer{renameLabels, DefaultLabelStyle}
var lowerCamelCustomizer = LabelCustomizer{renameLabels, SnakeToLowerCamel}
var upperCamelCustomizer = LabelCustomizer{renameLabels, SnakeToUpperCamel}

var defaultCustomize = defaultCustomizer.Customize
var lowerCamelCustomize = lowerCamelCustomizer.Customize
var upperCamelCustomize = upperCamelCustomizer.Customize

func TestEtsyFormatUnknownType(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = "UnknownMetricSetType"
	expectedMsg := ""

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, defaultCustomize, etsyMetricValue)
	assert.Error(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatPodContainerType(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypePodContainer
	expectedMsg := fmt.Sprintf("%s.node.%s.namespace.%s.pod.%s.container.%s.%s.%s:%v|g",
		etsyPrefix, etsyHostName, etsyNamespace, etsyPodName, etsyContainerName, expectedLowerCamelLabels, etsyMetricName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, lowerCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatSystemContainerType(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeSystemContainer
	expectedMsg := fmt.Sprintf("%s.node.%s.sys-container.%s.%s.%s:%v|g",
		etsyPrefix, etsyHostName, etsyContainerName, expectedUpperCamelLabels, etsyMetricName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, upperCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatPodType(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypePod
	expectedMsg := fmt.Sprintf("%s.node.%s.namespace.%s.pod.%s.%s.%s:%v|g",
		etsyPrefix, etsyHostName, etsyNamespace, etsyPodName, expectedLowerCamelLabels, etsyMetricName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, lowerCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatNamespaceType(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeNamespace
	expectedMsg := fmt.Sprintf("%s.namespace.%s.%s.%s:%v|g",
		etsyPrefix, etsyNamespace, expectedUpperCamelLabels, etsyMetricName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, upperCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatNodeType(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeNode
	expectedMsg := fmt.Sprintf("%s.node.%s.%s.%s:%v|g",
		etsyPrefix, etsyHostName, expectedLowerCamelLabels, etsyMetricName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, lowerCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatClusterType(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeCluster
	expectedMsg := fmt.Sprintf("%s.cluster.%s.%s:%v|g",
		etsyPrefix, expectedUpperCamelLabels, etsyMetricName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, upperCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatWithoutPrefix(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeCluster
	expectedMsg := fmt.Sprintf("cluster.%s.%s:%v|g",
		expectedLowerCamelLabels, etsyMetricName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format("", etsyMetricName, etsyLabels, lowerCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestEtsyFormatWithResource(t *testing.T) {
	etsyLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeCluster
	etsyLabels[core.LabelResourceID.Key] = etsyResourceName
	expectedMsg := fmt.Sprintf("%s.cluster.%s.%s.%s:%v|g",
		etsyPrefix, expectedUpperCamelLabels, etsyMetricName, etsyResourceName, etsyMetricValue.IntValue)

	formatter := NewEtsystatsdFormatter()
	assert.NotNil(t, formatter)

	msg, err := formatter.Format(etsyPrefix, etsyMetricName, etsyLabels, upperCamelCustomize, etsyMetricValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}
