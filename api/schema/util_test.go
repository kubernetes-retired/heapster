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
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/heapster/api/schema/info"
	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
	cadvisor "github.com/google/cadvisor/info/v1"
	fuzz "github.com/google/gofuzz"
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
	// Tests the flow of AddMetricToMap where the metric_name is present in the map
	var (
		metrics                   = make(map[string][]*info.MetricTimeseries)
		new_metric_name string    = "name_already_in_map"
		stamp           time.Time = time.Now()
		value           uint64    = 1234567890
	)

	old_metric := info.MetricTimeseries{
		Timestamp: time.Now(),
		Value:     value,
	}
	metrics[new_metric_name] = []*info.MetricTimeseries{&old_metric}

	assert := assert.New(t)
	assert.NoError(addMetricToMap(new_metric_name, stamp, value, &metrics))

	assert.Equal(metrics[new_metric_name][0], &old_metric)
	assert.Equal(metrics[new_metric_name][1].Timestamp, stamp)
	assert.Equal(metrics[new_metric_name][1].Value, value)
	assert.Len(metrics[new_metric_name], 2)
}

func TestAddMetricToMapNewKey(t *testing.T) {
	// Tests the flow of AddMetricToMap where the metric_name is not present in the map
	var (
		metrics                = make(map[string][]*info.MetricTimeseries)
		new_metric_name        = "name_not_in_map"
		stamp                  = time.Now()
		value           uint64 = 1234567890
	)
	assert := assert.New(t)
	assert.NoError(addMetricToMap(new_metric_name, stamp, value, &metrics))

	new_metric := &info.MetricTimeseries{
		Timestamp: stamp,
		Value:     value,
	}
	expected_slice := []*info.MetricTimeseries{new_metric}
	assert.Equal(metrics[new_metric_name], expected_slice)
}

func TestNewInfoType(t *testing.T) {
	// Tests both flows of the InfoType constructor
	var (
		metrics = make(map[string][]*info.MetricTimeseries)
		labels  = make(map[string]string)
	)
	new_metric := &info.MetricTimeseries{
		Timestamp: time.Unix(124124124, 0),
		Value:     uint64(5555),
	}
	metrics["test"] = []*info.MetricTimeseries{new_metric}
	labels["name"] = "test"
	assert := assert.New(t)

	// Invocation with no parameters
	new_infotype := newInfoType(nil, nil)
	assert.Empty(new_infotype.Metrics)
	assert.Empty(new_infotype.Labels)

	// Invocation with both parameters
	new_infotype = newInfoType(metrics, labels)
	assert.Equal(new_infotype.Metrics, metrics)
	assert.Equal(new_infotype.Labels, labels)
}

func TestUpdateInfoTypeEmptyCME(t *testing.T) {
	// Tests the error flow of UpdateInfoType
	// The ContainerElement passed as an argument has an empty Metrics slice
	// The empty slice error should propagate up from parseMetrics

	new_infotype := newInfoType(nil, nil)

	empty_ce := ContainerElementFactory([]*cache.ContainerMetricElement{})
	stamp, err := updateInfoType(&new_infotype, empty_ce)

	assert := assert.New(t)
	assert.Error(err)
	assert.Empty(new_infotype.Metrics)
	assert.Equal(stamp, time.Time{})
}

func TestUpdateInfoTypeNormal(t *testing.T) {
	// Tests the normal flows of UpdateInfoType
	// Calls updateInfoType for both empty, non-empty and erroneous arguments
	new_cme := cmeFactory()
	new_ce := ContainerElementFactory([]*cache.ContainerMetricElement{new_cme})
	new_infotype := newInfoType(nil, nil)

	// Invocation with empty infotype
	stamp, err := updateInfoType(&new_infotype, new_ce)

	assert := assert.New(t)
	assert.NoError(err)
	assert.Len(new_infotype.Metrics, 7) // Five stats total
	for _, metricSlice := range new_infotype.Metrics {
		assert.Len(metricSlice, 1) // 1 Metric per stat
	}

	assert.Empty(new_infotype.Labels)
	assert.NotEqual(stamp, time.Time{})

	// Invocation with an existing infotype as argument
	new_ce = ContainerElementFactory(nil)
	stamp, err = updateInfoType(&new_infotype, new_ce)

	assert.NoError(err)
	assert.Len(new_infotype.Metrics, 7) // Five stats total
	for _, metricSlice := range new_infotype.Metrics {
		assert.Len(metricSlice, 3) // 3 Metrics per stat
	}
	assert.Empty(new_infotype.Labels)
	assert.NotEqual(stamp, time.Time{})

	// Invocation with a nil argument -> error
	stamp, err = updateInfoType(nil, new_ce)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})

	// Invocation with a nil argument -> error
	stamp, err = updateInfoType(&new_infotype, nil)
	assert.Error(err)
	assert.Equal(stamp, time.Time{})
}

func TestParseMetricsError(t *testing.T) {
	// Tests the error flows of ParseMetrics

	// Invoke parseMetrics with a nil argument
	metrics, err := parseMetrics(nil)
	assert := assert.New(t)
	assert.Error(err)
	assert.Nil(metrics)

	// Invoke parseMetrics with a nil element in the CME slice
	empty_cme := emptyCmeFactory()
	metrics, err = parseMetrics([]*cache.ContainerMetricElement{empty_cme, nil})
	assert.Error(err)
	assert.Nil(metrics)
}

func TestParseMetricsNormal(t *testing.T) {
	// Tests the normal flow of ParseMetrics
	empty_cme := emptyCmeFactory()
	normal_cme := cmeFactory()

	// Normal Invocation with both empty and regular elements
	metrics, err := parseMetrics([]*cache.ContainerMetricElement{empty_cme, normal_cme})

	assert := assert.New(t)
	assert.NoError(err)
	assert.Len(metrics, 7)
	for key, metricSlice := range metrics {
		assert.Len(metricSlice, 1)
		metric := metricSlice[0]
		assert.Equal(metric.Timestamp, normal_cme.Stats.Timestamp)
		switch key {
		case "cpu/limit":
			assert.Equal(metric.Value, normal_cme.Spec.Cpu.Limit)
		case "memory/limit":
			assert.Equal(metric.Value, normal_cme.Spec.Memory.Limit)
		case "cpu/usage":
			assert.Equal(metric.Value, normal_cme.Stats.Cpu.Usage.Total)
		case "memory/usage":
			assert.Equal(metric.Value, normal_cme.Stats.Memory.Usage)
		case "memory/working":
			assert.Equal(metric.Value, normal_cme.Stats.Memory.WorkingSet)
		default:
			// Filesystem or error
			if strings.Contains(key, "limit") {
				assert.Equal(metric.Value, normal_cme.Stats.Filesystem[0].Limit)
			} else if strings.Contains(key, "usage") {
				assert.Equal(metric.Value, normal_cme.Stats.Filesystem[0].Usage)
			} else {
				assert.True(false, "Unknown key in resulting metrics slice")
			}
		}
	}
}

func TestAddContainerToMap(t *testing.T) {
	// Tests all the flows of AddContainerToMap
	new_map := make(map[string]*info.ContainerInfo)

	// Invocation where the ContainerInfo needs to be created
	cinfo := addContainerToMap("new_container", &new_map)

	assert := assert.New(t)
	assert.NotNil(cinfo)
	assert.NotNil(cinfo.Metrics)
	assert.NotNil(cinfo.Labels)
	assert.Equal(new_map["new_container"], cinfo)

	// Test Flow where ContainerInfo is already present
	new_cinfo := addContainerToMap("new_container", &new_map)
	assert.Equal(new_map["new_container"], new_cinfo)
	assert.Equal(cinfo, new_cinfo)
}

func cmeFactory() *cache.ContainerMetricElement {
	// Helper Function for testing
	// Generates a complete CME with Fuzzed data
	f := fuzz.New().NilChance(0).NumElements(1, 1)
	containerSpec := cadvisor.ContainerSpec{
		CreationTime:  time.Now(),
		HasCpu:        true,
		HasMemory:     true,
		HasNetwork:    true,
		HasFilesystem: true,
		HasDiskIo:     true,
	}
	var containerStats cadvisor.ContainerStats
	f.Fuzz(&containerStats)
	containerStats.Timestamp = time.Now()

	new_fs := cadvisor.FsStats{}
	f.Fuzz(&new_fs)
	new_fs.Device = "/dev/device1"
	containerStats.Filesystem = []cadvisor.FsStats{new_fs}

	return &cache.ContainerMetricElement{
		Spec:  &containerSpec,
		Stats: &containerStats,
	}
}

func emptyCmeFactory() *cache.ContainerMetricElement {
	// Test Helper Function
	// Generates a "useless" CME with no metrics
	f := fuzz.New().NilChance(0).NumElements(1, 1)
	containerSpec := cadvisor.ContainerSpec{
		CreationTime:  time.Now(),
		HasCpu:        false,
		HasMemory:     false,
		HasNetwork:    false,
		HasFilesystem: false,
		HasDiskIo:     false,
	}
	var containerStats cadvisor.ContainerStats
	f.Fuzz(&containerStats)
	containerStats.Timestamp = time.Now()

	return &cache.ContainerMetricElement{
		Spec:  &containerSpec,
		Stats: &containerStats,
	}
}

func ContainerElementFactory(cmes []*cache.ContainerMetricElement) *cache.ContainerElement {
	// Test Helper Function
	// Generates a ContainerElement object from an (optional) slice of CMEs
	var metrics []*cache.ContainerMetricElement
	if cmes == nil {
		// If the argument is nil, generate two CMEs
		metrics = []*cache.ContainerMetricElement{cmeFactory(), cmeFactory()}
	} else {
		metrics = cmes
	}
	metadata := cache.Metadata{
		Name:      "test",
		Namespace: "default",
		UID:       "123123123",
		Hostname:  "testhost",
		Labels:    make(map[string]string),
	}
	new_ce := cache.ContainerElement{
		Metadata: metadata,
		Metrics:  metrics,
	}
	return &new_ce
}
