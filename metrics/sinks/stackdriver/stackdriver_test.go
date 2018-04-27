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

package stackdriver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"k8s.io/heapster/metrics/core"
)

var (
	testProjectId = "test-project-id"
	zone          = "europe-west1-c"

	sink = &StackdriverSink{
		project:           testProjectId,
		heapsterZone:      zone,
		stackdriverClient: nil,
	}

	commonLabels     = map[string]string{}
	containerLabels  = map[string]string{"type": "pod_container"}
	podLabels        = map[string]string{"type": "pod"}
	nodeLabels       = map[string]string{"type": "node"}
	nodeDaemonLabels = map[string]string{"type": "sys_container", "container_name": "kubelet"}
)

func generateIntMetric(value int64) core.MetricValue {
	return core.MetricValue{
		ValueType: core.ValueInt64,
		IntValue:  value,
	}
}

func generateLabeledIntMetric(value int64, labels map[string]string, name string) core.LabeledMetric {
	return core.LabeledMetric{
		Name:        name,
		Labels:      labels,
		MetricValue: generateIntMetric(value),
	}
}

func generateFloatMetric(value float64) core.MetricValue {
	return core.MetricValue{
		ValueType:  core.ValueFloat,
		FloatValue: value,
	}
}

func deepCopy(source map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range source {
		result[k] = v
	}
	return result
}

// Test TranslateMetric

func testLegacyTranslateMetric(as *assert.Assertions, value int64, name string, labels map[string]string, expectedName string) *monitoringpb.TypedValue {
	metricValue := generateIntMetric(value)
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateMetric(timestamp, labels, name, metricValue, collectionStartTime)

	as.NotNil(ts)
	as.Equal(ts.Metric.Type, expectedName)
	as.Equal(len(ts.Points), 1)
	return ts.Points[0].Value
}

func testTranslateMetric(as *assert.Assertions, value core.MetricValue, labels map[string]string, name string, expectedName string) *monitoringpb.TypedValue {
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)
	entityCreateTime := timestamp.Add(-time.Second)

	ts := sink.TranslateMetric(timestamp, labels, name, value, collectionStartTime, entityCreateTime)

	as.NotNil(ts)
	as.Equal(ts.Metric.Type, expectedName)
	as.Equal(len(ts.Points), 1)
	return ts.Points[0].Value
}

func testTranslateLabeledMetric(as *assert.Assertions, labels map[string]string, metric core.LabeledMetric, expectedName string) *monitoringpb.TypedValue {
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.TranslateLabeledMetric(timestamp, labels, metric, collectionStartTime)

	as.NotNil(ts)
	as.Equal(ts.Metric.Type, expectedName)
	as.Equal(len(ts.Points), 1)
	return ts.Points[0].Value
}

func TestTranslateUptime(t *testing.T) {
	as := assert.New(t)
	value := testLegacyTranslateMetric(as, 30000, "uptime", commonLabels,
		"container.googleapis.com/container/uptime")

	as.Equal(30.0, value.GetDoubleValue())
}

func TestTranslateCpuUsage(t *testing.T) {
	as := assert.New(t)
	value := testLegacyTranslateMetric(as, 3600000000000, "cpu/usage", commonLabels,
		"container.googleapis.com/container/cpu/usage_time")

	as.Equal(3600.0, value.GetDoubleValue())
}

func TestTranslateCpuLimit(t *testing.T) {
	as := assert.New(t)
	value := testLegacyTranslateMetric(as, 2000, "cpu/limit", commonLabels,
		"container.googleapis.com/container/cpu/reserved_cores")

	as.Equal(2.0, value.GetDoubleValue())
}

func TestTranslateMemoryLimitNode(t *testing.T) {
	metricValue := generateIntMetric(2048)
	name := "memory/limit"
	timestamp := time.Now()

	labels := deepCopy(commonLabels)
	labels["type"] = core.MetricSetTypeNode

	ts := sink.LegacyTranslateMetric(timestamp, labels, name, metricValue, timestamp)
	var expected *monitoringpb.TimeSeries = nil

	as := assert.New(t)
	as.Equal(ts, expected)
}

func TestTranslateMemoryLimitPod(t *testing.T) {
	as := assert.New(t)
	labels := deepCopy(commonLabels)
	labels["type"] = core.MetricSetTypePod
	value := testLegacyTranslateMetric(as, 2048, "memory/limit", labels,
		"container.googleapis.com/container/memory/bytes_total")

	as.Equal(int64(2048), value.GetInt64Value())
}

func TestTranslateMemoryNodeAllocatable(t *testing.T) {
	as := assert.New(t)
	value := testLegacyTranslateMetric(as, 2048, "memory/node_allocatable", commonLabels,
		"container.googleapis.com/container/memory/bytes_total")

	as.Equal(int64(2048), value.GetInt64Value())
}

func TestTranslateMemoryUsedEvictable(t *testing.T) {
	metricValue := generateIntMetric(100)
	name := "memory/bytes_used"
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateMetric(timestamp, commonLabels, name, metricValue, collectionStartTime)

	as := assert.New(t)
	as.Equal(ts.Metric.Type, "container.googleapis.com/container/memory/bytes_used")
	as.Equal(len(ts.Points), 1)
	as.Equal(ts.Points[0].Value.GetInt64Value(), int64(100))
	as.Equal(ts.Metric.Labels["memory_type"], "evictable")
}

func TestTranslateMemoryUsedNonEvictable(t *testing.T) {
	metricValue := generateIntMetric(200)
	name := core.MetricMemoryWorkingSet.MetricDescriptor.Name
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateMetric(timestamp, commonLabels, name, metricValue, collectionStartTime)

	as := assert.New(t)
	as.Equal(ts.Metric.Type, "container.googleapis.com/container/memory/bytes_used")
	as.Equal(len(ts.Points), 1)
	as.Equal(ts.Points[0].Value.GetInt64Value(), int64(200))
	as.Equal(ts.Metric.Labels["memory_type"], "non-evictable")
}

func TestTranslateMemoryMajorPageFaults(t *testing.T) {
	metricValue := generateIntMetric(20)
	name := "memory/major_page_faults"
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateMetric(timestamp, commonLabels, name, metricValue, collectionStartTime)

	as := assert.New(t)
	as.Equal(ts.Metric.Type, "container.googleapis.com/container/memory/page_fault_count")
	as.Equal(len(ts.Points), 1)
	as.Equal(ts.Points[0].Value.GetInt64Value(), int64(20))
	as.Equal(ts.Metric.Labels["fault_type"], "major")
}

func TestTranslateMemoryMinorPageFaults(t *testing.T) {
	metricValue := generateIntMetric(42)
	name := "memory/minor_page_faults"
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateMetric(timestamp, commonLabels, name, metricValue, collectionStartTime)

	as := assert.New(t)
	as.Equal(ts.Metric.Type, "container.googleapis.com/container/memory/page_fault_count")
	as.Equal(len(ts.Points), 1)
	as.Equal(ts.Points[0].Value.GetInt64Value(), int64(42))
	as.Equal(ts.Metric.Labels["fault_type"], "minor")
}

// Test LegacyTranslateLabeledMetric

func TestTranslateFilesystemUsage(t *testing.T) {
	metric := core.LabeledMetric{
		MetricValue: generateIntMetric(10000),
		Labels: map[string]string{
			core.LabelResourceID.Key: "resource id",
		},
		Name: "filesystem/usage",
	}
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateLabeledMetric(timestamp, commonLabels, metric, collectionStartTime)

	as := assert.New(t)
	as.Equal(ts.Metric.Type, "container.googleapis.com/container/disk/bytes_used")
	as.Equal(len(ts.Points), 1)
	as.Equal(ts.Points[0].Value.GetInt64Value(), int64(10000))
}

func TestTranslateFilesystemLimit(t *testing.T) {
	metric := core.LabeledMetric{
		MetricValue: generateIntMetric(30000),
		Labels: map[string]string{
			core.LabelResourceID.Key: "resource id",
		},
		Name: "filesystem/limit",
	}
	timestamp := time.Now()
	collectionStartTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateLabeledMetric(timestamp, commonLabels, metric, collectionStartTime)

	as := assert.New(t)
	as.Equal(ts.Metric.Type, "container.googleapis.com/container/disk/bytes_total")
	as.Equal(len(ts.Points), 1)
	as.Equal(ts.Points[0].Value.GetInt64Value(), int64(30000))
}

func testTranslateAcceleratorMetric(t *testing.T, sourceName string, stackdriverName string) {
	value := int64(12345678)
	make := "nvidia"
	model := "Tesla P100"
	acceleratorID := "GPU-deadbeef-1234-5678-90ab-feedfacecafe"

	metric := generateLabeledIntMetric(
		value,
		map[string]string{
			core.LabelAcceleratorMake.Key:  make,
			core.LabelAcceleratorModel.Key: model,
			core.LabelAcceleratorID.Key:    acceleratorID,
		},
		sourceName)

	timestamp := time.Now()
	createTime := timestamp.Add(-time.Second)

	ts := sink.LegacyTranslateLabeledMetric(timestamp, commonLabels, metric, createTime)

	as := assert.New(t)
	as.Equal(stackdriverName, ts.Metric.Type)
	as.Equal(1, len(ts.Points))
	as.Equal(value, ts.Points[0].Value.GetInt64Value())
	as.Equal(make, ts.Metric.Labels["make"])
	as.Equal(model, ts.Metric.Labels["model"])
	as.Equal(acceleratorID, ts.Metric.Labels["accelerator_id"])
}

func TestTranslateAcceleratorMetrics(t *testing.T) {
	testTranslateAcceleratorMetric(t, "accelerator/memory_total", "container.googleapis.com/container/accelerator/memory_total")
	testTranslateAcceleratorMetric(t, "accelerator/memory_used", "container.googleapis.com/container/accelerator/memory_used")
	testTranslateAcceleratorMetric(t, "accelerator/duty_cycle", "container.googleapis.com/container/accelerator/duty_cycle")
}

// Test PreprocessMemoryMetrics

func TestComputeDerivedMetrics(t *testing.T) {
	as := assert.New(t)

	metricSet := &core.MetricSet{
		MetricValues: map[string]core.MetricValue{
			core.MetricMemoryUsage.MetricDescriptor.Name:           generateIntMetric(128),
			core.MetricMemoryWorkingSet.MetricDescriptor.Name:      generateIntMetric(32),
			core.MetricMemoryPageFaults.MetricDescriptor.Name:      generateIntMetric(42),
			core.MetricMemoryMajorPageFaults.MetricDescriptor.Name: generateIntMetric(29),
		},
	}

	computedMetrics := sink.computeDerivedMetrics(metricSet)

	as.Equal(int64(96), computedMetrics.MetricValues["memory/bytes_used"].IntValue)
	as.Equal(int64(13), computedMetrics.MetricValues["memory/minor_page_faults"].IntValue)
}

func TestTranslateMetricSet(t *testing.T) {
	as := assert.New(t)

	containerUptime := testTranslateMetric(as, generateIntMetric(1000), containerLabels, "uptime", "kubernetes.io/container/uptime")
	containerCpuUsage := testTranslateMetric(as, generateIntMetric(2000000000), containerLabels, "cpu/usage", "kubernetes.io/container/cpu/core_usage_time")
	containerCpuRequest := testTranslateMetric(as, generateIntMetric(3000), containerLabels, "cpu/request", "kubernetes.io/container/cpu/request_cores")
	containerCpuLimit := testTranslateMetric(as, generateIntMetric(4000), containerLabels, "cpu/limit", "kubernetes.io/container/cpu/limit_cores")
	containerMemoryUsage := testTranslateMetric(as, generateIntMetric(5), containerLabels, "memory/bytes_used", "kubernetes.io/container/memory/used_bytes")
	containerMemoryRequest := testTranslateMetric(as, generateIntMetric(6), containerLabels, "memory/request", "kubernetes.io/container/memory/request_bytes")
	containerMemoryLimit := testTranslateMetric(as, generateIntMetric(7), containerLabels, "memory/limit", "kubernetes.io/container/memory/limit_bytes")
	containerEphemeralStorageUsage := testTranslateMetric(as, generateIntMetric(5), containerLabels, "ephemeral_storage/usage", "kubernetes.io/container/ephemeral_storage/used_bytes")
	containerEphemeralStorageRequest := testTranslateMetric(as, generateIntMetric(6), containerLabels, "ephemeral_storage/request", "kubernetes.io/container/ephemeral_storage/request_bytes")
	containerEphemeralStorageLimit := testTranslateMetric(as, generateIntMetric(7), containerLabels, "ephemeral_storage/limit", "kubernetes.io/container/ephemeral_storage/limit_bytes")
	containerRestartCount := testTranslateMetric(as, generateIntMetric(8), containerLabels, "restart_count", "kubernetes.io/container/restart_count")
	podNetworkBytesRx := testTranslateMetric(as, generateIntMetric(9), podLabels, "network/rx", "kubernetes.io/pod/network/received_bytes_count")
	podNetworkBytesTx := testTranslateMetric(as, generateIntMetric(10), podLabels, "network/tx", "kubernetes.io/pod/network/sent_bytes_count")
	podVolumeTotal := testTranslateLabeledMetric(as, podLabels, generateLabeledIntMetric(11, map[string]string{}, "filesystem/limit"), "kubernetes.io/pod/volume/total_bytes")
	podVolumeUsed := testTranslateLabeledMetric(as, podLabels, generateLabeledIntMetric(12, map[string]string{}, "filesystem/usage"), "kubernetes.io/pod/volume/used_bytes")
	nodeCpuUsage := testTranslateMetric(as, generateIntMetric(13000000000), nodeLabels, "cpu/usage", "kubernetes.io/node/cpu/core_usage_time")
	nodeCpuTotal := testTranslateMetric(as, generateFloatMetric(14000), nodeLabels, "cpu/node_capacity", "kubernetes.io/node/cpu/total_cores")
	nodeCpuAllocatable := testTranslateMetric(as, generateFloatMetric(15000), nodeLabels, "cpu/node_allocatable", "kubernetes.io/node/cpu/allocatable_cores")
	nodeMemoryUsage := testTranslateMetric(as, generateIntMetric(16), nodeLabels, "memory/bytes_used", "kubernetes.io/node/memory/used_bytes")
	nodeMemoryTotal := testTranslateMetric(as, generateFloatMetric(17), nodeLabels, "memory/node_capacity", "kubernetes.io/node/memory/total_bytes")
	nodeMemoryAllocatable := testTranslateMetric(as, generateFloatMetric(18), nodeLabels, "memory/node_allocatable", "kubernetes.io/node/memory/allocatable_bytes")
	nodeNetworkBytesRx := testTranslateMetric(as, generateIntMetric(19), nodeLabels, "network/rx", "kubernetes.io/node/network/received_bytes_count")
	nodeNetworkBytesTx := testTranslateMetric(as, generateIntMetric(20), nodeLabels, "network/tx", "kubernetes.io/node/network/sent_bytes_count")
	nodeDaemonCpuUsage := testTranslateMetric(as, generateIntMetric(21000000000), nodeDaemonLabels, "cpu/usage", "kubernetes.io/node_daemon/cpu/core_usage_time")
	nodeDaemonMemoryUsage := testTranslateMetric(as, generateIntMetric(22), nodeDaemonLabels, "memory/bytes_used", "kubernetes.io/node_daemon/memory/used_bytes")

	as.Equal(float64(1), containerUptime.GetDoubleValue())
	as.Equal(float64(2), containerCpuUsage.GetDoubleValue())
	as.Equal(float64(3), containerCpuRequest.GetDoubleValue())
	as.Equal(float64(4), containerCpuLimit.GetDoubleValue())
	as.Equal(int64(5), containerMemoryUsage.GetInt64Value())
	as.Equal(int64(6), containerMemoryRequest.GetInt64Value())
	as.Equal(int64(7), containerMemoryLimit.GetInt64Value())
	as.Equal(int64(8), containerRestartCount.GetInt64Value())
	as.Equal(int64(9), podNetworkBytesRx.GetInt64Value())
	as.Equal(int64(10), podNetworkBytesTx.GetInt64Value())
	as.Equal(int64(11), podVolumeTotal.GetInt64Value())
	as.Equal(int64(12), podVolumeUsed.GetInt64Value())
	as.Equal(float64(13), nodeCpuUsage.GetDoubleValue())
	as.Equal(float64(14), nodeCpuTotal.GetDoubleValue())
	as.Equal(float64(15), nodeCpuAllocatable.GetDoubleValue())
	as.Equal(int64(16), nodeMemoryUsage.GetInt64Value())
	as.Equal(int64(17), nodeMemoryTotal.GetInt64Value())
	as.Equal(int64(18), nodeMemoryAllocatable.GetInt64Value())
	as.Equal(int64(19), nodeNetworkBytesRx.GetInt64Value())
	as.Equal(int64(20), nodeNetworkBytesTx.GetInt64Value())
	as.Equal(float64(21), nodeDaemonCpuUsage.GetDoubleValue())
	as.Equal(int64(22), nodeDaemonMemoryUsage.GetInt64Value())
	as.Equal(int64(5), containerEphemeralStorageUsage.GetInt64Value())
	as.Equal(int64(6), containerEphemeralStorageRequest.GetInt64Value())
	as.Equal(int64(7), containerEphemeralStorageLimit.GetInt64Value())
}
