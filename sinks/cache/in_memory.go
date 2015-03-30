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

package cache

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
)

type entry struct {
	timestamp time.Time
	data      interface{}
}

// TODO: Consider using cadvisor's in memory storage instead.
type timeStore struct {
	buffer *list.List
	rwLock sync.RWMutex
}

func (ts *timeStore) Put(timestamp time.Time, data interface{}) error {
	if data == nil {
		return fmt.Errorf("cannot store nil data")
	}
	ts.rwLock.Lock()
	defer ts.rwLock.Unlock()
	if ts.buffer.Len() == 0 {
		glog.V(5).Infof("put pushfront: %v, %v", timestamp, data)
		ts.buffer.PushFront(entry{timestamp: timestamp, data: data})
		return nil
	}
	for elem := ts.buffer.Front(); elem != nil; elem = elem.Next() {
		if timestamp.After(elem.Value.(entry).timestamp) {
			glog.V(5).Infof("put insert before: %v, %v, %v", elem, timestamp, data)
			ts.buffer.InsertBefore(entry{timestamp: timestamp, data: data}, elem)
			return nil
		}
	}
	glog.V(5).Infof("put pushback: %v, %v", timestamp, data)
	ts.buffer.PushBack((entry{timestamp: timestamp, data: data}))
	return nil
}

func (ts *timeStore) GetAll() []interface{} {
	ts.rwLock.RLock()
	defer ts.rwLock.RUnlock()
	result := []interface{}{}
	for elem := ts.buffer.Back(); elem != nil; elem = elem.Prev() {
		result = append(result, elem.Value.(entry).data)
	}

	return result
}

func (ts *timeStore) Get(start, end time.Time) ([]interface{}, error) {
	ts.rwLock.RLock()
	defer ts.rwLock.RUnlock()
	if ts.buffer.Len() == 0 {
		return nil, nil
	}
	if !end.After(start) {
		return nil, fmt.Errorf("End time %v is not after Start time %v", end, start)
	}
	result := []interface{}{}
	for elem := ts.buffer.Back(); elem != nil; elem = elem.Prev() {
		entry := elem.Value.(entry)
		if entry.timestamp.Before(start) || entry.timestamp.After(end) {
			break
		}
		result = append(result, entry.data)
	}
	return result, nil
}

func (ts *timeStore) Last() interface{} {
	ts.rwLock.RLock()
	defer ts.rwLock.RUnlock()
	if ts.buffer.Len() == 0 {
		return nil
	}
	elem := ts.buffer.Front()
	return elem.Value.(entry).data
}

func (ts *timeStore) Delete(start, end time.Time) error {
	ts.rwLock.Lock()
	defer ts.rwLock.Unlock()
	if ts.buffer.Len() == 0 {
		return nil
	}
	if !end.After(start) {
		return fmt.Errorf("End time %v is not after Start time %v", end, start)
	}
	elem := ts.buffer.Back()
	for elem != nil {
		entry := elem.Value.(entry)
		if entry.timestamp.Before(start) || entry.timestamp.After(end) {
			break
		}
		oldElem := elem
		elem = elem.Prev()
		ts.buffer.Remove(oldElem)
	}
	return nil
}

func NewTimeStore() TimeStore {
	return &timeStore{
		rwLock: sync.RWMutex{},
		buffer: list.New(),
	}
}
