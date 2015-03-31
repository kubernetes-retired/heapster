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

package api

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	cadvisor "github.com/google/cadvisor/info/v1"
	"github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyInput(t *testing.T) {
	timeseries, err := NewMetricDecoder().Timeseries(source_api.AggregateData{})
	assert.NoError(t, err)
	assert.Empty(t, timeseries)
}

func getLabelsAsString(labels map[string]string) string {
	output := make([]string, 0, len(labels))
	for key, value := range labels {
		output = append(output, fmt.Sprintf("%s:%s", key, value))
	}

	// Sort to produce a stable output.
	sort.Strings(output)
	return strings.Join(output, ",")
}

func getContainer(name string) source_api.Container {
	f := fuzz.New().NumElements(1, 1).NilChance(0)
	containerSpec := cadvisor.ContainerSpec{
		CreationTime:  time.Unix(12345, 0),
		HasCpu:        true,
		HasMemory:     true,
		HasNetwork:    true,
		HasFilesystem: true,
		HasDiskIo:     true,
	}
	containerStats := make([]*cadvisor.ContainerStats, 1)
	f.Fuzz(&containerStats)
	return source_api.Container{
		Name:  name,
		Spec:  containerSpec,
		Stats: containerStats,
	}
}

func getFsStats(input source_api.AggregateData) map[string]int64 {
	expectedFsStats := map[string]int64{}
	for _, cont := range input.Containers {
		for _, stat := range cont.Stats {
			for _, fs := range stat.Filesystem {
				expectedFsStats[fs.Device] = int64(fs.Usage)
			}
		}
	}
	return expectedFsStats
}

func TestRealInput(t *testing.T) {
	containers := []source_api.Container{
		getContainer("container1"),
	}
	pods := []source_api.Pod{
		{
			Name:       "pod1",
			ID:         "123",
			Hostname:   "1.2.3.4",
			Status:     "Running",
			Containers: containers,
		},
		{
			Name:       "pod2",
			ID:         "123",
			Hostname:   "1.2.3.5",
			Status:     "Running",
			Containers: containers,
		},
	}
	input := source_api.AggregateData{
		Pods:       pods,
		Containers: containers,
		Machine:    containers,
	}
	timeseries, err := NewMetricDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)

	expectedFsStats := getFsStats(input)

	metrics := make(map[string][]Timeseries)
	for index := range timeseries {
		series, ok := metrics[timeseries[index].SeriesName]
		if !ok {
			series = make([]Timeseries, 0)
		}
		series = append(series, timeseries[index])
		metrics[timeseries[index].SeriesName] = series
	}

	for index := range statMetrics {
		series, ok := metrics[statMetrics[index].MetricDescriptor.Name]
		require.True(t, ok)
		for innerIndex, entry := range series {
			assert.Equal(t, statMetrics[index].MetricDescriptor.Name, series[innerIndex].SeriesName)
			stats := containers[0].Stats[0]
			switch entry.SeriesName {
			case "uptime":
				_, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				// TODO(saadali):
				// Do not test the value of uptime. It is the delta between the container creation time
				// and the current time. Since both the creation time and the current time were not fixed,
				// this check was flaky. The correct way to do the test is to set the creation time of the
				// container to something fixed (modified the fuzzer to do that), and (todo) modify
				// supported_metrics to use an interface for time.Since, so that it can be replaced at test
				// time to compare to a fixed current time.
			case "cpu/usage":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Cpu.Usage.Total, value)
			case "memory/usage":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Memory.Usage, value)
			case "memory/working_set":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Memory.WorkingSet, value)
			case "memory/page_faults":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Memory.ContainerData.Pgfault, value)
			case "network/rx":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Network.RxBytes, value)
			case "network/rx_errors":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Network.RxErrors, value)
			case "network/tx":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Network.TxBytes, value)
			case "network/tx_errors":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Network.TxErrors)
			case "filesystem/usage":
				value, ok := entry.Points[0].Value.(int64)
				require.True(t, ok)
				name, ok := entry.Points[0].Labels[labelResourceID]
				require.True(t, ok)
				assert.Equal(t, value, expectedFsStats[name])
			default:
				t.Errorf("unexpected metric type")
			}
		}
	}
}

func TestFuzzInput(t *testing.T) {
	var input source_api.AggregateData
	fuzz.New().Fuzz(&input)
	timeseries, err := NewMetricDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
}

func TestPodLabelsProcessing(t *testing.T) {
	podLabels := map[string]string{"key1": "value1", "key2": "value2"}
	pods := []source_api.Pod{
		{
			Name:       "pod1",
			ID:         "123",
			Hostname:   "1.2.3.4",
			Status:     "Running",
			Labels:     podLabels,
			Containers: []source_api.Container{getContainer("blah")},
		},
	}

	expectedLabels := map[string]string{
		labelPodId:         "123",
		labelPodName:       "pod1",
		labelLabels:        getLabelsAsString(podLabels),
		labelHostname:      "1.2.3.4",
		labelContainerName: "blah",
	}
	input := source_api.AggregateData{
		Pods: pods,
	}
	timeseries, err := NewMetricDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	// ignore ResourceID label.
	for _, entry := range timeseries {
		for name, value := range expectedLabels {
			assert.Equal(t, value, entry.Points[0].Labels[name])
		}
	}
}

func TestContainerLabelsProcessing(t *testing.T) {
	expectedLabels := map[string]string{
		labelHostname:      "1.2.3.4",
		labelContainerName: "blah",
	}
	container := getContainer("blah")
	container.Hostname = "1.2.3.4"

	input := source_api.AggregateData{
		Containers: []source_api.Container{container},
	}
	timeseries, err := NewMetricDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	// ignore ResourceID label.
	for _, entry := range timeseries {
		for name, value := range expectedLabels {
			assert.Equal(t, value, entry.Points[0].Labels[name])
		}
	}
}
