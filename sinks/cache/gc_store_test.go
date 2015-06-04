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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGC(t *testing.T) {
	gcStore := NewGCStore(NewTimeStore(), time.Microsecond, time.Microsecond)
	now := time.Now()
	assert := assert.New(t)
	for i := 0; i < 100; i++ {
		assert.NoError(gcStore.Put(time.Now(), struct{}{}))
	}
	time.Sleep(time.Second)
	// Perform a put to invoke GC.
	assert.NoError(gcStore.Put(time.Now(), struct{}{}))
	data := gcStore.Get(now, time.Now())
	assert.Len(data, 0)
}

func TestGCDetail(t *testing.T) {
	gcStore := NewGCStore(NewTimeStore(), time.Hour, time.Microsecond)
	now := time.Now()
	assert := assert.New(t)
	for i := 0; i < 20; i++ {
		timestamp := now.Add(-time.Hour + (time.Duration(i) * time.Minute))
		assert.NoError(gcStore.Put(timestamp, struct{}{}))
	}
	data := gcStore.Get(time.Time{}, time.Now())
	assert.NotEmpty(data)
}

func TestLongGC(t *testing.T) {
	gcStore := NewGCStore(NewTimeStore(), time.Hour, time.Microsecond)
	now := time.Now()
	assert := assert.New(t)
	for i := 0; i < 200; i++ {
		assert.NoError(gcStore.Put(time.Now(), i))
	}
	data := gcStore.Get(now, time.Now())
	assert.Equal(200, len(data))
}
