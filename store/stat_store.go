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
	"sync"
	"time"

	"github.com/google/cadvisor/summary"
)

const (
	Fiftieth    float64 = 0.50
	Ninetieth   float64 = 0.90
	NinetyFifth float64 = 0.95
)

// StatStore is a TimeStore that can also fetch stats over its own data.
// It assumes that the underlying TimeStore uses TimePoint values of type
// uint64.
type StatStore interface {
	TimeStore

	// Last returns the last TimePoint
	Last() *TimePoint
	// GetAverage gets the average value of the data
	Average() uint64
	// GetMax gets the max value of the data
	Max() uint64
	// Percentile gets the specified Nth percentile of the data
	Percentile(n float64) uint64
}

type statStore struct {
	ts TimeStore

	cacheLock         sync.Mutex
	validCache        bool
	cachedAverage     float64
	cachedMax         uint64
	cachedFiftieth    uint64
	cachedNinetieth   uint64
	cachedNinetyFifth uint64
}

func (s *statStore) Put(tp TimePoint) error {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	s.validCache = false
	return s.ts.Put(tp)
}

func (s *statStore) Get(start, end time.Time) []TimePoint {
	return s.ts.Get(start, end)
}

func (s *statStore) Delete(start, end time.Time) error {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	s.validCache = false
	return s.ts.Delete(start, end)
}

func (s *statStore) getAll() []TimePoint {
	zeroTime := time.Time{}
	return s.ts.Get(zeroTime, zeroTime)
}

func (s *statStore) Last() *TimePoint {
	all := s.getAll()
	if len(all) < 1 {
		return nil
	}
	// To not give the impression that this allows the caller to change
	// the last value in the underlying data structure, return the address
	// of a copy
	last := all[len(all)-1]
	return &last
}

func (s *statStore) fillCache() {
	if s.validCache {
		return
	}

	s.validCache = true
	s.cachedAverage = 0
	s.cachedMax = 0
	s.cachedFiftieth = 0
	s.cachedNinetieth = 0
	s.cachedNinetyFifth = 0

	all := s.getAll()
	if len(all) < 1 {
		return
	}

	inc := make(summary.Uint64Slice, 0, len(all))
	for _, tp := range all {
		inc = append(inc, tp.Value.(uint64))
	}
	acc := uint64(0)
	for _, u := range inc {
		acc += u
	}
	s.cachedAverage = float64(acc) / float64(len(inc))
	s.cachedFiftieth = inc.GetPercentile(Fiftieth)
	s.cachedNinetieth = inc.GetPercentile(Ninetieth)
	s.cachedNinetyFifth = inc.GetPercentile(NinetyFifth)
	// inc is sorted in ascending order by GetPercentile
	// TODO(afein || mvdan): sort explicitly only once
	s.cachedMax = inc[len(inc)-1]
}

func (s *statStore) Average() uint64 {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	s.fillCache()
	return uint64(s.cachedAverage)
}

func (s *statStore) Max() uint64 {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	s.fillCache()
	return s.cachedMax
}

func (s *statStore) Percentile(n float64) uint64 {
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock()
	s.fillCache()
	return s.cachedNinetyFifth
}

func NewStatStore(store TimeStore) StatStore {
	return &statStore{
		ts: store,
	}
}
