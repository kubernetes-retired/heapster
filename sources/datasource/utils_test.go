// Copyright 2014 Google Inc. All Rights Reserved.
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

// This file implements a cadvisor datasource, that collects metrics from an instance
// of cadvisor runing on a specific host.

package datasource

import (
	"testing"
	"time"

	cadvisor "github.com/google/cadvisor/info/v1"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
)

func TestWithFuzzInput(t *testing.T) {
	for i := 0; i < 1000; i++ {
		testWithFuzzInternal()
	}
}

func testWithFuzzInternal() {
	inputStats := []*cadvisor.ContainerStats{}
	fuzz.New().Fuzz(&inputStats)
	_ = sampleContainerStats(inputStats, time.Now(), time.Nanosecond, false)
	_ = sampleContainerStats(inputStats, time.Now(), time.Nanosecond, true)
}

func TestWithNoDownsampling(t *testing.T) {
	start := time.Now()
	resolution := time.Second
	inputStats := []*cadvisor.ContainerStats{
		{
			Timestamp: start,
		},
		{
			Timestamp: start.Add(resolution),
		},
		{
			Timestamp: start.Add(resolution * 2),
		},
		{
			Timestamp: start.Add(resolution * 3),
		},
	}
	output := sampleContainerStats(inputStats, start, resolution, false)
	assert.Equal(t, output, inputStats)
}

func TestWithDownsampling(t *testing.T) {
	start := time.Now()
	inputStats := []*cadvisor.ContainerStats{}
	for i := 1; i <= 4; i++ {
		inputStats = append(inputStats, &cadvisor.ContainerStats{Timestamp: start.Add(time.Second * time.Duration(i))})
	}

	output := sampleContainerStats(inputStats, start, time.Second*2, false)
	assert.Len(t, output, 2)
}

func TestWithLargeDownsampling(t *testing.T) {
	start := time.Now()
	inputStats := []*cadvisor.ContainerStats{}
	for i := 1; i <= 100; i++ {
		inputStats = append(inputStats, &cadvisor.ContainerStats{Timestamp: start.Add(time.Second * time.Duration(i))})
	}
	output := sampleContainerStats(inputStats, start, time.Minute, false)
	assert.Len(t, output, 2)
}

func getTime(h, m, s int) time.Time {
	return time.Date(2015, time.July, 10, h, m, s, 0, time.UTC)
}

func TestWithNoDownsamplingAndAligning(t *testing.T) {
	resolution := time.Minute
	start := getTime(10, 0, 5)
	inputStats := []*cadvisor.ContainerStats{
		{
			Timestamp: getTime(10, 1, 5),
		},
		{
			Timestamp: getTime(10, 2, 5),
		},
	}
	outputStats := []*cadvisor.ContainerStats{
		{
			Timestamp: getTime(10, 1, 0),
		},
		{
			Timestamp: getTime(10, 2, 0),
		},
	}
	output := sampleContainerStats(inputStats, start, resolution, true)
	assert.Equal(t, output, outputStats)
}

func TestWithDownsamplingAndAligning(t *testing.T) {
	resolution := time.Minute
	start := getTime(10, 0, 5)
	inputStats := []*cadvisor.ContainerStats{
		{
			Timestamp: getTime(10, 0, 1),
		},
		{
			Timestamp: getTime(10, 0, 10),
		},
		{
			Timestamp: getTime(10, 1, 1),
		},
		{
			Timestamp: getTime(10, 1, 10),
		},
		{
			Timestamp: getTime(10, 2, 1),
		},
		{
			Timestamp: getTime(10, 2, 10),
		},
	}
	outputStats := []*cadvisor.ContainerStats{
		{
			Timestamp: getTime(10, 1, 0),
		},
		{
			Timestamp: getTime(10, 2, 0),
		},
	}
	output := sampleContainerStats(inputStats, start, resolution, true)
	assert.Equal(t, output, outputStats)
}

func TestWithAligningAndAlignedStart(t *testing.T) {
	resolution := time.Minute
	start := getTime(10, 0, 0)
	inputStats := []*cadvisor.ContainerStats{
		{
			Timestamp: getTime(10, 0, 1),
		},
		{
			Timestamp: getTime(10, 0, 10),
		},
		{
			Timestamp: getTime(10, 1, 1),
		},
		{
			Timestamp: getTime(10, 1, 10),
		},
		{
			Timestamp: getTime(10, 2, 1),
		},
		{
			Timestamp: getTime(10, 2, 10),
		},
	}
	outputStats := []*cadvisor.ContainerStats{
		{
			Timestamp: getTime(10, 0, 0),
		},
		{
			Timestamp: getTime(10, 1, 0),
		},
		{
			Timestamp: getTime(10, 2, 0),
		},
	}
	output := sampleContainerStats(inputStats, start, resolution, true)
	assert.Equal(t, output, outputStats)
}
