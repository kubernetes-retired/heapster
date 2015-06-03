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

	"github.com/GoogleCloudPlatform/heapster/Godeps/_workspace/src/github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestInitialization(t *testing.T) {
	store := NewTimeStore()
	data := store.Get(time.Now().Add(-time.Minute), time.Now())
	assert.Empty(t, len(data), time.Now())
}

func TestNilInsert(t *testing.T) {
	store := NewTimeStore()
	assert.Error(t, store.Put(time.Now(), nil))
}

func TestInsert(t *testing.T) {
	store := NewTimeStore()
	now := time.Now()
	assert.NoError(t, store.Put(now, 2))
	assert.NoError(t, store.Put(now.Add(-time.Second), 1))
	assert.NoError(t, store.Put(now.Add(time.Second), 3))
	assert.NoError(t, store.Put(now.Add(-2*time.Second), 0))
	actual := store.Get(time.Time{}, now.Add(time.Second))
	require.Len(t, actual, 4)
	for i := 0; i < len(actual); i++ {
		assert.Equal(t, i, actual[i].(int))
	}
	actual = store.Get(time.Time{}, time.Time{})
	require.Len(t, actual, 4)
	for i := 0; i < len(actual); i++ {
		assert.Equal(t, i, actual[i].(int))
	}
}

func TestDelete(t *testing.T) {
	store := NewTimeStore()
	now := time.Now()
	assert.NoError(t, store.Put(now, 2))
	assert.NoError(t, store.Put(now.Add(-time.Second), 1))
	assert.NoError(t, store.Put(now.Add(time.Second), 3))
	assert.NoError(t, store.Delete(now.Add(-time.Second), now))
	actual := store.Get(now.Add(-time.Second), time.Time{})
	require.Len(t, actual, 1)
	assert.Equal(t, 3, actual[0].(int))
}
