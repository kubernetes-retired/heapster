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
	"time"

	"github.com/GoogleCloudPlatform/heapster/store"
)

// latestTimestamp returns its largest time.Time argument
func latestTimestamp(first time.Time, second time.Time) time.Time {
	if first.After(second) {
		return first
	}
	return second
}

// newInfoType is an InfoType Constructor, which returns a new InfoType.
// Initial fields for the new InfoType can be provided as arguments.
// A nil argument results in a newly-allocated map for that field.
func newInfoType(metrics map[string]*store.TimeStore, labels map[string]string) InfoType {
	if metrics == nil {
		metrics = make(map[string]*store.TimeStore)
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	return InfoType{
		Metrics: metrics,
		Labels:  labels,
	}
}

// addContainerToMap creates or finds a ContainerInfo element under a map[string]*ContainerInfo
func addContainerToMap(container_name string, dict map[string]*ContainerInfo) *ContainerInfo {
	var container_ptr *ContainerInfo

	if val, ok := dict[container_name]; ok {
		// A container already exists under that name, return the address
		container_ptr = val
	} else {
		container_ptr = &ContainerInfo{
			InfoType: newInfoType(nil, nil),
		}
		dict[container_name] = container_ptr
	}
	return container_ptr
}

// addTimePoints adds the values of two TimePoints as uint64.
// addTimePoints returns a new TimePoint with the added Value fields
// and the Timestamp of the first TimePoint.
func addTimePoints(tp1 store.TimePoint, tp2 store.TimePoint) store.TimePoint {
	return store.TimePoint{
		Timestamp: tp1.Timestamp,
		Value:     tp1.Value.(uint64) + tp2.Value.(uint64),
	}
}

// popTPSlice pops the first element of a TimePoint Slice, removing it from the slice.
// popTPSlice receives a *[]TimePoint and returns its first element.
func popTPSlice(tps_ptr *[]store.TimePoint) *store.TimePoint {
	if tps_ptr == nil {
		return nil
	}
	tps := *tps_ptr
	if len(tps) == 0 {
		return nil
	}
	res := tps[0]
	if len(tps) == 1 {
		(*tps_ptr) = tps[0:0]
	}
	(*tps_ptr) = tps[1:]
	return &res
}

// addMatchingTimeseries performs addition over two timeseries with unique timestamps.
// addMatchingTimeseries returns a []TimePoint of the resulting aggregated.
// Assumes time-descending order of both []TimePoint parameters and the return slice.
func addMatchingTimeseries(left []store.TimePoint, right []store.TimePoint) []store.TimePoint {
	var cur_left *store.TimePoint
	var cur_right *store.TimePoint
	result := []store.TimePoint{}

	// Merge timeseries into result until either one is empty
	cur_left = popTPSlice(&left)
	cur_right = popTPSlice(&right)
	for cur_left != nil && cur_right != nil {
		if cur_left.Timestamp.Equal(cur_right.Timestamp) {
			result = append(result, addTimePoints(*cur_left, *cur_right))
			cur_left = popTPSlice(&left)
			cur_right = popTPSlice(&right)
		} else if cur_left.Timestamp.After(cur_right.Timestamp) {
			result = append(result, *cur_left)
			cur_left = popTPSlice(&left)
		} else {
			result = append(result, *cur_right)
			cur_right = popTPSlice(&right)
		}
	}
	if cur_left == nil && cur_right != nil {
		result = append(result, *cur_right)
	} else if cur_left != nil && cur_right == nil {
		result = append(result, *cur_left)
	}

	// Append leftover elements from non-empty timeseries
	if len(left) > 0 {
		result = append(result, left...)
	} else if len(right) > 0 {
		result = append(result, right...)
	}

	return result
}
