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
	"time"
)

// DayStore holds an 1-hour StatStore whose average is taken every one hour.
// DayStore also holds a 1-day StatStore where the individual sub-averages are placed.
type DayStore struct {
	Hour          StatStore
	Day           StatStore
	DayMaxima     TimeStore
	created       time.Time
	lastUpdate    time.Time
	dayResolution time.Duration
}

// DayMax returns the maximum value that has occured over the past Day.
// A Timestore is used to ensure that the maximum reflects only the last day's values.
// TODO(afein): cache the curMax until a new value is inserted in DayMaxima.
func (ds *DayStore) DayMax() uint64 {
	zeroTime := time.Time{}
	allTPs := ds.DayMaxima.Get(zeroTime, zeroTime)
	curMax := uint64(0)
	for _, tp := range allTPs {
		if newVal := tp.Value.(uint64); newVal > curMax {
			curMax = newVal
		}
	}

	hourMax := ds.Hour.Max()
	if hourMax > curMax {
		curMax = hourMax
	}
	return curMax
}

func (ds *DayStore) Get(start time.Time, end time.Time) []TimePoint {
	return ds.Hour.Get(start, end)
}

func (ds *DayStore) Put(tp TimePoint) error {
	err := ds.Hour.Put(tp)
	if err != nil {
		return err
	}

	// Mark the created field if this is the first TimePoint
	if !ds.created.After(time.Time{}) {
		ds.created = tp.Timestamp
	}

	if tp.Timestamp.Add(-ds.dayResolution).After(ds.lastUpdate) {
		ds.Day.Put(TimePoint{
			Timestamp: tp.Timestamp,
			Value:     ds.Hour.Average(),
		})
		ds.DayMaxima.Put(TimePoint{
			Timestamp: tp.Timestamp,
			Value:     ds.Hour.Max(),
		})
		ds.lastUpdate = tp.Timestamp
	}

	return nil
}

func NewDayStore(res time.Duration) DayStore {
	return DayStore{
		Hour:          NewStatStore(NewGCStore(NewCMAStore(), time.Hour)),
		Day:           NewStatStore(NewGCStore(NewCMAStore(), 24*time.Hour)),
		DayMaxima:     NewGCStore(NewCMAStore(), 24*time.Hour),
		created:       time.Time{},
		lastUpdate:    time.Time{},
		dayResolution: res,
	}
}
