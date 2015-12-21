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

package metricsink

import (
	"sync"
	"time"

	"k8s.io/heapster/metrics/core"
)

// A simple in-memory storage for metrics. It divides metrics into 2 categories
// * metrics that need to be stored for couple minutes.
// * metrics that need to be stored for longer time (15 min, 1 hour).
// The user of this struct needs to decide what are the long-stored metrics uprfront.
type MetricSink struct {
	// Request can come from other threads
	lock sync.Mutex

	// list of metrics that will be stored for up to X seconds
	longStoreMetrics   []string
	longStoreDuration  time.Duration
	shortStoreDuration time.Duration

	// Stores full DataBatch with all metrics and labels
	shortStore []*core.DataBatch
	// Stores timmed DataBatches with selected metrics and no labels
	longStore []*core.DataBatch
}

type TimestampedMetricValue struct {
	core.MetricValue
	Timestamp time.Time
}

func (this *MetricSink) Name() string {
	return "MetricSink"
}

func (this *MetricSink) Stop() {
	// Do nothing.
}

func (this *MetricSink) ExportData(batch *core.DataBatch) {
	trimmed := this.trimDataBatch(batch)

	this.lock.Lock()
	defer this.lock.Unlock()

	now := time.Now()
	// TODO: add sorting
	this.longStore = append(popOld(this.longStore, now.Add(-this.longStoreDuration)), trimmed)
	this.shortStore = append(popOld(this.shortStore, now.Add(-this.shortStoreDuration)), batch)
}

func (this *MetricSink) GetMetric(metricName string, keys []string, start, end time.Time) map[string][]TimestampedMetricValue {
	this.lock.Lock()
	defer this.lock.Unlock()

	var storeToUse []*core.DataBatch = nil
	for _, longStoreMetric := range this.longStoreMetrics {
		if longStoreMetric == metricName {
			storeToUse = this.longStore
		}
	}
	if storeToUse == nil {
		storeToUse = this.shortStore
	}

	result := make(map[string][]TimestampedMetricValue)

	for _, batch := range storeToUse {
		// Inclusive start and end.
		if !batch.Timestamp.Before(start) && !batch.Timestamp.After(end) {
			for _, key := range keys {
				metricSet, found := batch.MetricSets[key]
				if !found {
					continue
				}
				metricValue, found := metricSet.MetricValues[metricName]
				if !found {
					continue
				}
				keyResult, found := result[key]
				if !found {
					keyResult = make([]TimestampedMetricValue, 0)
				}
				keyResult = append(keyResult, TimestampedMetricValue{
					Timestamp:   batch.Timestamp,
					MetricValue: metricValue,
				})
				result[key] = keyResult
			}
		}
	}
	return result
}

func (this *MetricSink) GetMetricNames(key string) []string {
	this.lock.Lock()
	defer this.lock.Unlock()

	metricNames := make(map[string]bool)
	for _, batch := range this.longStore {
		if set, found := batch.MetricSets[key]; found {
			for key := range set.MetricValues {
				metricNames[key] = true
			}
		}
	}
	for _, batch := range this.shortStore {
		if set, found := batch.MetricSets[key]; found {
			for key := range set.MetricValues {
				metricNames[key] = true
			}
		}
	}
	result := make([]string, 0, len(metricNames))
	for key := range metricNames {
		result = append(result, key)
	}
	return result
}

func (this *MetricSink) trimDataBatch(batch *core.DataBatch) *core.DataBatch {
	result := core.DataBatch{
		Timestamp:  batch.Timestamp,
		MetricSets: make(map[string]*core.MetricSet),
	}
	for metricSetKey, metricSet := range batch.MetricSets {
		trimmedMetricSet := core.MetricSet{
			MetricValues: make(map[string]core.MetricValue),
		}
		for _, metricName := range this.longStoreMetrics {
			metricValue, found := metricSet.MetricValues[metricName]
			if found {
				trimmedMetricSet.MetricValues[metricName] = metricValue
			}
		}
		if len(trimmedMetricSet.MetricValues) > 0 {
			result.MetricSets[metricSetKey] = &trimmedMetricSet
		}
	}
	return &result
}

func popOld(storage []*core.DataBatch, cutoffTime time.Time) []*core.DataBatch {
	result := make([]*core.DataBatch, 0)
	for _, batch := range storage {
		if batch.Timestamp.After(cutoffTime) {
			result = append(result, batch)
		}
	}
	return result
}

func NewMetricSink(shortStoreDuration, longStoreDuration time.Duration, longStoreMetrics []string) *MetricSink {
	return &MetricSink{
		longStoreMetrics:   longStoreMetrics,
		longStoreDuration:  longStoreDuration,
		shortStoreDuration: shortStoreDuration,
		longStore:          make([]*core.DataBatch, 0),
		shortStore:         make([]*core.DataBatch, 0),
	}
}
