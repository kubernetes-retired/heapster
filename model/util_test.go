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

package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/heapster/store"
)

// newTimePoint is a helper function that constructs TimePoints for other unit tests.
func newTimePoint(stamp time.Time, val interface{}) store.TimePoint {
	return store.TimePoint{
		Timestamp: stamp,
		Value:     val,
	}
}

// TestLatestsTimestamp tests all flows of latestTimeStamp.
func TestLatestTimestamp(t *testing.T) {
	assert := assert.New(t)
	past := time.Unix(1434212566, 0)
	future := time.Unix(1434212800, 0)
	assert.Equal(latestTimestamp(past, future), future)
	assert.Equal(latestTimestamp(future, past), future)
	assert.Equal(latestTimestamp(future, future), future)
}

// TestNewInfoType tests both flows of the InfoType constructor.
func TestNewInfoType(t *testing.T) {
	var (
		metrics = make(map[string]*store.TimeStore)
		labels  = make(map[string]string)
	)
	new_store := newTimeStore()
	metrics["test"] = &new_store
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

// TestAddContainerToMap tests all the flows of addContainerToMap.
func TestAddContainerToMap(t *testing.T) {
	new_map := make(map[string]*ContainerInfo)

	// First Call: A new ContainerInfo is created
	cinfo := addContainerToMap("new_container", new_map)

	assert := assert.New(t)
	assert.NotNil(cinfo)
	assert.NotNil(cinfo.Metrics)
	assert.NotNil(cinfo.Labels)
	assert.Equal(new_map["new_container"], cinfo)

	// Second Call: A ContainerInfo is already available for that key
	new_cinfo := addContainerToMap("new_container", new_map)
	assert.Equal(new_map["new_container"], new_cinfo)
	assert.Equal(cinfo, new_cinfo)
}

// TestAddTimePoints tests all flows of addTimePoints.
func TestAddTimePoints(t *testing.T) {
	var (
		tp1 = store.TimePoint{
			Timestamp: time.Unix(1434212800, 0),
			Value:     uint64(500),
		}
		tp2 = store.TimePoint{
			Timestamp: time.Unix(1434212805, 0),
			Value:     uint64(700),
		}
		assert = assert.New(t)
	)
	new_tp := addTimePoints(tp1, tp2)
	assert.Equal(new_tp.Timestamp, tp1.Timestamp)
	assert.Equal(new_tp.Value.(uint64), tp1.Value.(uint64)+tp2.Value.(uint64))
}

// TestPopTPSlice tests all flows of PopTPSlice.
func TestPopTPSlice(t *testing.T) {
	var (
		tp1 = store.TimePoint{
			Timestamp: time.Now(),
			Value:     uint64(3),
		}
		tp2 = store.TimePoint{
			Timestamp: time.Now(),
			Value:     uint64(4),
		}
		assert = assert.New(t)
	)

	// Invocations with no popped element
	assert.Nil(popTPSlice(nil))
	assert.Nil(popTPSlice(&[]store.TimePoint{}))

	// Invocation with one element
	tps := []store.TimePoint{tp1}
	res := popTPSlice(&tps)
	assert.Equal(*res, tp1)
	assert.Empty(tps)

	// Invocation with two elements
	tps = []store.TimePoint{tp1, tp2}
	res = popTPSlice(&tps)
	assert.Equal(*res, tp1)
	assert.Len(tps, 1)
	assert.Equal(tps[0], tp2)
}

// TestAddMatchingTimeseries tests the normal flow of addMatchingTimeseries.
func TestAddMatchingTimeseries(t *testing.T) {
	var (
		tp11 = store.TimePoint{
			Timestamp: time.Unix(1434212800, 0),
			Value:     uint64(4),
		}
		tp21 = store.TimePoint{
			Timestamp: time.Unix(1434212805, 0),
			Value:     uint64(9),
		}
		tp31 = store.TimePoint{
			Timestamp: time.Unix(1434212807, 0),
			Value:     uint64(10),
		}
		tp41 = store.TimePoint{
			Timestamp: time.Unix(1434212850, 0),
			Value:     uint64(533),
		}

		tp12 = store.TimePoint{
			Timestamp: time.Unix(1434212800, 0),
			Value:     uint64(4),
		}
		tp22 = store.TimePoint{
			Timestamp: time.Unix(1434212806, 0),
			Value:     uint64(8),
		}
		tp32 = store.TimePoint{
			Timestamp: time.Unix(1434212807, 0),
			Value:     uint64(11),
		}
		tp42 = store.TimePoint{
			Timestamp: time.Unix(1434212851, 0),
			Value:     uint64(534),
		}
		tp52 = store.TimePoint{
			Timestamp: time.Unix(1434212859, 0),
			Value:     uint64(538),
		}
		assert = assert.New(t)
	)
	// Invocation with 1+1 data points
	tps1 := []store.TimePoint{tp11}
	tps2 := []store.TimePoint{tp12}
	new_ts := addMatchingTimeseries(tps1, tps2)
	assert.Len(new_ts, 1)
	assert.Equal(new_ts[0].Timestamp, tp11.Timestamp)
	assert.Equal(new_ts[0].Value, uint64(8))

	// Invocation with 3+1 data points
	tps1 = []store.TimePoint{tp52, tp42, tp11}
	tps2 = []store.TimePoint{tp12}
	new_ts = addMatchingTimeseries(tps1, tps2)
	assert.Len(new_ts, 3)
	assert.Equal(new_ts[0], tp52)
	assert.Equal(new_ts[1], tp42)
	assert.Equal(new_ts[2].Timestamp, tp11.Timestamp)
	assert.Equal(new_ts[2].Value, tp11.Value.(uint64)+tp12.Value.(uint64))

	// Invocation with 4+5 data points
	tps1 = []store.TimePoint{tp41, tp31, tp21, tp11}
	tps2 = []store.TimePoint{tp52, tp42, tp32, tp22, tp12}
	new_ts = addMatchingTimeseries(tps1, tps2)
	assert.Len(new_ts, 7)
	assert.Equal(new_ts[0], tp52)
	assert.Equal(new_ts[1], tp42)
	assert.Equal(new_ts[2], tp41)
	assert.Equal(new_ts[3].Timestamp, tp31.Timestamp)
	assert.Equal(new_ts[3].Value, uint64(21))
	assert.Equal(new_ts[4], tp22)
	assert.Equal(new_ts[5], tp21)
	assert.Equal(new_ts[6].Timestamp, tp11.Timestamp)
	assert.Equal(new_ts[6].Value, uint64(8))
}

// TestAddMatchingTimeseriesEmpty tests the alternate flows of addMatchingTimeseries.
// Three permutations of empty parameters are tested.
func TestAddMatchingTimeseriesEmpty(t *testing.T) {
	var (
		tp12 = store.TimePoint{
			Timestamp: time.Unix(1434212800, 0),
			Value:     uint64(4),
		}
		tp22 = store.TimePoint{
			Timestamp: time.Unix(1434212806, 0),
			Value:     uint64(8),
		}
		tp32 = store.TimePoint{
			Timestamp: time.Unix(1434212807, 0),
			Value:     uint64(11),
		}
		assert = assert.New(t)
	)
	empty_tps := []store.TimePoint{}
	tps := []store.TimePoint{tp12, tp22, tp32}

	// First call: first argument is empty
	new_ts := addMatchingTimeseries(empty_tps, tps)
	assert.Equal(new_ts, tps)

	// Second call: second argument is empty
	new_ts = addMatchingTimeseries(tps, empty_tps)
	assert.Equal(new_ts, tps)

	// Third call: both arguments are empty
	new_ts = addMatchingTimeseries(empty_tps, empty_tps)
	assert.Equal(new_ts, empty_tps)
}
