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

package v1

import (
	"time"
)

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     uint64    `json:"value"`
}

type MetricResult struct {
	Metrics         []MetricPoint `json:"metrics"`
	LatestTimestamp time.Time     `json:"latestTimestamp"`
}

type Stats struct {
	Average    uint64 `json:"average"`
	Percentile uint64 `json:"percentile"`
	Max        uint64 `json:"max"`
}

type ExternalStatBundle struct {
	Minute Stats `json:"minute"`
	Hour   Stats `json:"hour"`
	Day    Stats `json:"day"`
}

type StatsResponse struct {
	// Uptime in seconds
	Uptime uint64                        `json:"uptime"`
	Stats  map[string]ExternalStatBundle `json:"stats"`
}
