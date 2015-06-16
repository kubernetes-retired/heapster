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

	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	cadvisor "github.com/google/cadvisor/info/v1"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeContainerCreationTime = 12345
	fakeCurrentTime           = 12350
)

func TestEmptyInput(t *testing.T) {
	timeseries, err := NewDecoder().Timeseries(source_api.AggregateData{})
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
		CreationTime:  time.Unix(fakeContainerCreationTime, 0),
		HasCpu:        true,
		HasMemory:     true,
		HasNetwork:    true,
		HasFilesystem: true,
		HasDiskIo:     true,
		Cpu: cadvisor.CpuSpec{
			Limit: 100,
		},
		Memory: cadvisor.MemorySpec{
			Limit: 100,
		},
	}
	containerStats := make([]*cadvisor.ContainerStats, 1)
	f.Fuzz(&containerStats)
	return source_api.Container{
		Name:  name,
		Spec:  containerSpec,
		Stats: containerStats,
	}
}

type fsStats struct {
	limit int64
	usage int64
}

func getFsStats(input source_api.AggregateData) map[string]fsStats {
	expectedFsStats := map[string]fsStats{}
	for _, cont := range input.Containers {
		for _, stat := range cont.Stats {
			for _, fs := range stat.Filesystem {
				expectedFsStats[fs.Device] = fsStats{int64(fs.Limit), int64(fs.Usage)}
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
			PodMetadata: source_api.PodMetadata{
				Name:      "pod1",
				ID:        "123",
				Namespace: "test",
				Hostname:  "1.2.3.4",
				Status:    "Running",
			},
			Containers: containers,
		},
		{
			PodMetadata: source_api.PodMetadata{
				Name:      "pod2",
				ID:        "123",
				Namespace: "test",
				Hostname:  "1.2.3.5",
				Status:    "Running",
			},
			Containers: containers,
		},
	}
	input := source_api.AggregateData{
		Pods:       pods,
		Containers: containers,
		Machine:    containers,
	}
	timeseries, err := NewDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)

	expectedFsStats := getFsStats(input)

	metrics := make(map[string][]sink_api.Timeseries)
	for index := range timeseries {
		series, ok := metrics[timeseries[index].Point.Name]
		if !ok {
			series = make([]sink_api.Timeseries, 0)
		}
		series = append(series, timeseries[index])
		metrics[timeseries[index].Point.Name] = series
	}

	statMetrics := sink_api.SupportedStatMetrics()
	for index := range statMetrics {
		series, ok := metrics[statMetrics[index].MetricDescriptor.Name]
		require.True(t, ok)
		for innerIndex, entry := range series {
			assert.Equal(t, statMetrics[index].MetricDescriptor, *series[innerIndex].MetricDescriptor)
			spec := containers[0].Spec
			stats := containers[0].Stats[0]
			switch entry.Point.Name {
			case "uptime":
				_, ok := entry.Point.Value.(int64)
				require.True(t, ok)
			case "cpu/usage":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Cpu.Usage.Total, value)
			case "memory/usage":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Memory.Usage, value)
			case "memory/working_set":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Memory.WorkingSet, value)
			case "memory/page_faults":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Memory.ContainerData.Pgfault, value)
			case "memory/major_page_faults":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Memory.ContainerData.Pgmajfault, value)
			case "network/rx":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Network.RxBytes, value)
			case "network/rx_errors":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Network.RxErrors, value)
			case "network/tx":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Network.TxBytes, value)
			case "network/tx_errors":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, stats.Network.TxErrors, value)
			case "filesystem/usage":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				name, ok := entry.Point.Labels[sink_api.LabelResourceID.Key]
				require.True(t, ok)
				assert.Equal(t, expectedFsStats[name].usage, value)
			case "cpu/limit":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				expected := (spec.Cpu.Limit * 1000) / 1024
				assert.Equal(t, expected, value)
			case "memory/limit":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, spec.Memory.Limit, value)
			case "filesystem/limit":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				name, ok := entry.Point.Labels[sink_api.LabelResourceID.Key]
				require.True(t, ok)
				assert.Equal(t, expectedFsStats[name].limit, value)
			default:
				t.Errorf("unexpected metric type")
			}
		}
	}
}

func TestFuzzInput(t *testing.T) {
	var input source_api.AggregateData
	fuzz.New().Fuzz(&input)
	timeseries, err := NewDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
}

func TestPodLabelsProcessing(t *testing.T) {
	podLabels := map[string]string{"key1": "value1", "key2": "value2"}
	pods := []source_api.Pod{
		{
			PodMetadata: source_api.PodMetadata{
				Name:      "pod1",
				ID:        "123",
				Namespace: "test",
				Hostname:  "1.2.3.4",
				Status:    "Running",
				Labels:    podLabels,
			},
			Containers: []source_api.Container{getContainer("blah")},
		},
	}

	expectedLabels := map[string]string{
		sink_api.LabelPodId.Key:         "123",
		sink_api.LabelPodNamespace.Key:  "test",
		sink_api.LabelLabels.Key:        getLabelsAsString(podLabels),
		sink_api.LabelHostname.Key:      "1.2.3.4",
		sink_api.LabelContainerName.Key: "blah",
	}
	input := source_api.AggregateData{
		Pods: pods,
	}
	timeseries, err := NewDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	// ignore ResourceID label.
	for _, entry := range timeseries {
		for name, value := range expectedLabels {
			assert.Equal(t, entry.Point.Labels[name], value)
		}
	}
}

func TestContainerLabelsProcessing(t *testing.T) {
	expectedLabels := map[string]string{
		sink_api.LabelHostname.Key:      "1.2.3.4",
		sink_api.LabelContainerName.Key: "blah",
	}
	container := getContainer("blah")
	container.Hostname = "1.2.3.4"

	input := source_api.AggregateData{
		Containers: []source_api.Container{container},
	}
	timeseries, err := NewDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	// ignore ResourceID label.
	for _, entry := range timeseries {
		for name, value := range expectedLabels {
			assert.Equal(t, entry.Point.Labels[name], value)
		}
	}
}
