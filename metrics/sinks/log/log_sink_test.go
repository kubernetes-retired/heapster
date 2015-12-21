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

package logsink

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/heapster/metrics/core"
)

func TestSimpleWrite(t *testing.T) {
	now := time.Now()
	batch := core.DataBatch{
		Timestamp:  now,
		MetricSets: make(map[string]*core.MetricSet),
	}
	batch.MetricSets["pod1"] = &core.MetricSet{
		Labels: map[string]string{"bzium": "hocuspocus"},
		MetricValues: map[string]core.MetricValue{
			"m1": core.MetricValue{
				ValueType:  core.ValueInt64,
				MetricType: core.MetricGauge,
				IntValue:   31415,
			},
		},
	}
	log := batchToString(&batch)

	assert.True(t, strings.Contains(log, "31415"))
	assert.True(t, strings.Contains(log, "m1"))
	assert.True(t, strings.Contains(log, "bzium"))
	assert.True(t, strings.Contains(log, "hocuspocus"))
	assert.True(t, strings.Contains(log, "pod1"))
	assert.True(t, strings.Contains(log, fmt.Sprintf("%s", now)))
}
