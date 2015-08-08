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

package store

import (
	"container/list"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

func NewStatStore(epsilon uint64, resolution time.Duration, windowDuration uint8, supportedPercentiles []float64) *statStore {
	return &statStore{
		buffer:               list.New(),
		epsilon:              epsilon,
		resolution:           resolution,
		windowDuration:       windowDuration,
		supportedPercentiles: supportedPercentiles,
	}
}

// statStore is an in-memory rolling timeseries buffer that allows the extraction of -
// segments of the stored timeseries and derived stats.
// The tradeoff between space and precision can be configured through epsilon and resolution.
// statStore only retains information about the latest TimePoints that are within its
// specified duration.
// statStore is a TimeStore-like object that does not implement the TimeStore interface.
type statStore struct {
	// start is the start of the time window.
	start time.Time

	// buffer is a list of tpBucket that is sequenced in a time-descending order, meaning that
	// Front points to the latest tpBucket and Back to the oldest one.
	buffer *list.List

	// A RWMutex guards all operations on the underlying buffer.
	sync.RWMutex

	// epsilon is the acceptable error difference for the storage of TimePoints.
	// Increasing epsilon decreases the memory usage of the statStore, at the cost of precision.
	// The precision of max is not affected.
	epsilon uint64

	// resolution is the standardized duration between points in the statStore.
	// the Get operation returns TimePoints at every multiple of resolution,
	// even if TimePoints have not been Put for all such times.
	resolution time.Duration

	// windowDuration is the maximum number of time resolutions that is stored in the statStore
	// e.g. windowDuration 60, with a resolution of time.Minute represents an 1-hour window
	windowDuration uint8

	// tpCount is the number of TimePoints that are represented in the statStore.
	// if tpCount is equal to windowDuration, then the statStore window is considered full.
	tpCount uint8

	// lastPut maintains the state of values inserted within the last resolution.
	// When values of a later resolution are added to the statStore, lastPut is flushed to
	// the appropriate tpBucket.
	lastPut putState

	// suportedPercentiles is a slice of values from (0,1) that represents the percentiles
	// that are calculated by the statStore.
	supportedPercentiles []float64

	validCache bool

	// cachedAverage, cachedMax and cachedPercentiles are the cached derived stats that -
	// are exposed by the StatStore.
	cachedAverage     uint64
	cachedMax         uint64
	cachedPercentiles []uint64
}

// tpBucket is a bucket that represents a set of consecutive TimePoints with different
// timestamps whose values differ less than epsilon.
// tpBucket essentially represents a time window with a constant value.
type tpBucket struct {
	// count is the number of TimePoints represented in the tpBucket.
	count uint8

	// value is the approximate value of all in the tpBucket, +- epsilon.
	value uint64

	// max is the maximum value of all TimePoints that have been used to generate the tpBucket.
	max uint64

	// maxIdx is the number of resolutions after the start time where the max value is located.
	maxIdx uint8
}

// putState is a structure that maintains context of the values in the resolution that is currently
// being inserted.
// Assumes that Puts are performed in a time-ascending order.
type putState struct {
	actualCount uint16
	average     float64
	max         uint64
	stamp       time.Time
}

func (ss *statStore) Put(tp TimePoint) error {
	ss.Lock()
	defer ss.Unlock()

	ss.validCache = false

	// Flatten timestamp to the last multiple of resolution
	ts := tp.Timestamp.Truncate(ss.resolution)

	lastPutTime := ss.lastPut.stamp

	// Handle the case where the buffer and lastPut are both empty
	if lastPutTime.Equal(time.Time{}) {
		ss.resetLastPut(ts, tp.Value.(uint64))
		return nil
	}

	if ts.Before(lastPutTime) {
		// Ignore TimePoints with Timestamps in the past
		return fmt.Errorf("the provided timepoint has a timestamp in the past")
	}

	if ts.Equal(lastPutTime) {
		// update lastPut with the new TimePoint
		newVal := tp.Value.(uint64)
		if newVal > ss.lastPut.max {
			ss.lastPut.max = newVal
		}
		oldAvg := ss.lastPut.average
		n := float64(ss.lastPut.actualCount)
		ss.lastPut.average = (float64(newVal) + (n * oldAvg)) / (n + 1)
		ss.lastPut.actualCount++
		return nil
	}

	// new point is in the future, lastPut needs to be flushed to the statStore.

	// Determine how many resolutions in the future the new point is at.
	// The statStore always represents values up until 1 resolution from lastPut.
	numRes := uint8(0)
	curr := ts
	for curr.After(ss.lastPut.stamp) {
		curr = curr.Add(-ss.resolution)
		numRes++
	}

	// Create a new bucket if the buffer is empty
	if ss.buffer.Front() == nil {
		ss.newBucket(numRes)
		ss.resetLastPut(ts, tp.Value.(uint64))
		for ss.tpCount > ss.windowDuration {
			ss.rewind()
		}
		return nil
	}

	lastEntry := ss.buffer.Front().Value.(tpBucket)
	lastAvg := ss.lastPut.average

	// Place lastPut in the latest bucket if the difference from its average
	// is less than epsilon
	if uint64(math.Abs(float64(lastEntry.value)-lastAvg)) < ss.epsilon {
		lastEntry.count += numRes
		ss.tpCount += numRes
		if ss.lastPut.max > lastEntry.max {
			lastEntry.max = ss.lastPut.max
			lastEntry.maxIdx = lastEntry.count - 1
		}
		// update in list
		ss.buffer.Front().Value = lastEntry
	} else {
		// Create a new bucket
		ss.newBucket(numRes)
	}

	// Delete the earliest represented TimePoints if the window is full
	for ss.tpCount > ss.windowDuration {
		ss.rewind()
	}

	ss.resetLastPut(ts, tp.Value.(uint64))
	return nil
}

// resetLastPut initializes the lastPut field of the statStore from a time and a value.
func (ss *statStore) resetLastPut(timestamp time.Time, value uint64) {
	ss.lastPut.stamp = timestamp
	ss.lastPut.actualCount = 1
	ss.lastPut.average = float64(value)
	ss.lastPut.max = value
}

// newBucket appends a new bucket to the statStore, using the values of lastPut.
// newBucket should be called BEFORE resetting lastPut.
// newBuckets are created by rounding up the lastPut average to the closest epsilon.
// The numRes parameter is the number of resolutions that this bucket will hold.
// numRes represents the number of resolutions from the newest TimePoint to the lastPut.
func (ss *statStore) newBucket(numRes uint8) {
	newEntry := tpBucket{
		count:  numRes,
		value:  ((uint64(ss.lastPut.average) / ss.epsilon) + 1) * ss.epsilon,
		max:    ss.lastPut.max,
		maxIdx: 0,
	}
	ss.buffer.PushFront(newEntry)
	ss.tpCount += numRes

	// If this was the first bucket, update ss.start
	if ss.start.Equal(time.Time{}) {
		ss.start = ss.lastPut.stamp
	}
}

// rewind deletes the oldest one resolution of data in the statStore.
func (ss *statStore) rewind() {
	firstElem := ss.buffer.Back()
	firstEntry := firstElem.Value.(tpBucket)
	// Decrement number of TimePoints in the earliest tpBucket
	firstEntry.count--
	// Decrement total number of TimePoints in the statStore
	ss.tpCount--
	if firstEntry.maxIdx == 0 {
		// The Max value was just removed, lose precision for other maxes in this bucket
		firstEntry.max = firstEntry.value
		firstEntry.maxIdx = firstEntry.count - 1
	} else {
		firstEntry.maxIdx--
	}
	if firstEntry.count == 0 {
		// Delete the entry if no TimePoints are represented any more
		ss.buffer.Remove(firstElem)
	} else {
		firstElem.Value = firstEntry
	}

	// Update the start time of the statStore
	ss.start = ss.start.Add(ss.resolution)
}

// Get generates a []TimePoint from the appropriate tpEntries.
// Get receives a start and end time as inputs.
func (ss *statStore) Get(start, end time.Time) []TimePoint {
	ss.RLock()
	defer ss.RUnlock()

	var result []TimePoint

	if ss.buffer.Len() == 0 || (start.After(end) && end.After(time.Time{})) {
		return result
	}

	skipped := 0
	for elem := ss.buffer.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(tpBucket)

		// calculate the start time of the entry
		offset := int(ss.tpCount) - skipped - int(entry.count)
		entryStart := ss.start.Add(time.Duration(offset) * ss.resolution)

		// ignore tpEntries later than the requested end time
		if end.After(time.Time{}) && entryStart.After(end) {
			skipped += int(entry.count)
			continue
		}

		// break if we have reached a tpBucket with no values before or equal to
		// the start time.
		if !entryStart.Add(time.Duration(entry.count-1) * ss.resolution).After(start) {
			break
		}

		// generate as many TimePoints as required from this bucket
		newSkip := 0
		for curr := 1; curr <= int(entry.count); curr++ {
			offset = int(ss.tpCount) - skipped - curr
			newStamp := ss.start.Add(time.Duration(offset) * ss.resolution)
			if end.After(time.Time{}) && newStamp.After(end) {
				continue
			}

			if newStamp.Before(start) {
				break
			}

			// this TimePoint is in (start, end), generate it
			newSkip++
			newTP := TimePoint{
				Timestamp: newStamp,
				Value:     entry.value,
			}
			result = append(result, newTP)
		}
		skipped += newSkip
	}
	return result

}

// Last returns the latest TimePoint represented in the statStore,
// or an error if the statStore is empty.
// Note: the lastPut field is not used, as we are not confident of its final value.
func (ss *statStore) Last() (TimePoint, error) {
	ss.RLock()
	defer ss.RUnlock()

	// Obtain latest tpBucket
	lastElem := ss.buffer.Front()
	if lastElem == nil {
		return TimePoint{}, fmt.Errorf("the statStore is empty")
	}

	lastEntry := lastElem.Value.(tpBucket)

	// create the latest TimePoint from lastEntry
	lastTP := TimePoint{
		Timestamp: ss.start.Add(time.Duration(ss.tpCount-1) * ss.resolution),
		Value:     lastEntry.value,
	}

	return lastTP, nil
}

// fillCache caches the average, max and percentiles of the statStore.
// assumes a write lock is taken by the caller.
func (ss *statStore) fillCache() {

	// calculate the average value
	sum := uint64(0)
	for elem := ss.buffer.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(tpBucket)
		sum += uint64(entry.count) * entry.value
	}
	ss.cachedAverage = sum / uint64(ss.tpCount)

	// calculate the max value
	curMax := uint64(0)
	for elem := ss.buffer.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(tpBucket)
		if entry.max > curMax {
			curMax = entry.max
		}
	}
	ss.cachedMax = curMax

	// sort all values in the statStore
	vals := []float64{}
	for elem := ss.buffer.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(tpBucket)
		for i := uint8(0); i < entry.count; i++ {
			vals = append(vals, float64(entry.value))
		}
	}
	sort.Float64s(vals)

	// calculate all percentiles
	ss.cachedPercentiles = []uint64{}
	for _, spc := range ss.supportedPercentiles {
		pcIdx := int(math.Trunc(spc * float64(ss.tpCount)))
		ss.cachedPercentiles = append(ss.cachedPercentiles, uint64(vals[pcIdx]))
	}

	ss.validCache = true
}

// Average performs a weighted average across all buckets, using the count of -
// resolutions at each bucket as the weight.
func (ss *statStore) Average() (uint64, error) {
	ss.RLock()
	defer ss.RUnlock()

	lastElem := ss.buffer.Front()
	if lastElem == nil {
		return uint64(0), fmt.Errorf("the statStore is empty")
	}

	if !ss.validCache {
		ss.fillCache()
	}
	return ss.cachedAverage, nil
}

// Max returns the maximum element currently in the statStore.
// Max does NOT consider the case where the maximum is in the last one minute.
func (ss *statStore) Max() (uint64, error) {
	ss.RLock()
	defer ss.RUnlock()

	if ss.buffer.Front() == nil {
		return uint64(0), fmt.Errorf("the statStore is empty")
	}

	if !ss.validCache {
		ss.fillCache()
	}
	return ss.cachedMax, nil
}

// Percentile returns the requested percentile from the statStore.
func (ss *statStore) Percentile(p float64) (uint64, error) {
	ss.RLock()
	defer ss.RUnlock()

	if ss.buffer.Front() == nil {
		return uint64(0), fmt.Errorf("the statStore is empty")
	}

	// Check if the specific percentile is supported
	found := false
	idx := 0
	for i, spc := range ss.supportedPercentiles {
		if p == spc {
			found = true
			idx = i
			break
		}
	}

	if !found {
		return uint64(0), fmt.Errorf("the requested percentile is not supported")
	}

	if !ss.validCache {
		ss.fillCache()
	}

	return ss.cachedPercentiles[idx], nil
}
