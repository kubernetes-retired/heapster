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

package schema

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/heapster/store"
	"github.com/stretchr/testify/assert"
)

// TestLatestsTimestamp tests all flows of latestTimeStamp
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
	new_store := store.NewGCStore(store.NewTimeStore(), time.Hour)
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
