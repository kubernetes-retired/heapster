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
		CreationTime:  time.Now(),
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
	timeseries, err := NewDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)

	metrics := make(map[string][]Timeseries)
	for index := range timeseries {
		series, ok := metrics[timeseries[index].Point.Name]
		if !ok {
			series = make([]Timeseries, 0)
		}
		series = append(series, timeseries[index])
		metrics[timeseries[index].Point.Name] = series
	}
	for index := range statMetrics {
		series, ok := metrics[statMetrics[index].MetricDescriptor.Name]
		require.True(t, ok)
		for innerIndex, entry := range series {
			assert.Equal(t, statMetrics[index].MetricDescriptor, *series[innerIndex].MetricDescriptor)
			spec := containers[0].Spec
			stats := containers[0].Stats[0]
			switch entry.Point.Name {
			case "uptime":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				expected := time.Since(spec.CreationTime).Nanoseconds() / time.Millisecond.Nanoseconds()
				assert.Equal(t, value, expected)
			case "cpu/usage":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Cpu.Usage.Total)
			case "memory/usage":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Memory.Usage)
			case "memory/working_set":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Memory.WorkingSet)
			case "memory/page_faults":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Memory.ContainerData.Pgfault)
			case "network/rx":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Network.RxBytes)
			case "network/rx_errors":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Network.RxErrors)
			case "network/tx":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Network.TxBytes)
			case "network/tx_errors":
				value, ok := entry.Point.Value.(int64)
				require.True(t, ok)
				assert.Equal(t, value, stats.Network.TxErrors)
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
		labelLabels:        getLabelsAsString(podLabels),
		labelHostname:      "1.2.3.4",
		labelContainerName: "blah",
	}
	input := source_api.AggregateData{
		Pods: pods,
	}
	timeseries, err := NewDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	for _, entry := range timeseries {
		assert.Equal(t, expectedLabels, entry.Point.Labels)
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
	timeseries, err := NewDecoder().Timeseries(input)
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	for _, entry := range timeseries {
		assert.Equal(t, expectedLabels, entry.Point.Labels)
	}
}
