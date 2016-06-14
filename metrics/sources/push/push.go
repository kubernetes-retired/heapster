// Copyright 2016 Google Inc. All Rights Reserved.
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

package push

import (
	"fmt"
	"sync"
	"time"

	. "k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/util"

	"github.com/golang/glog"
)

// for testing
var nowFunc = time.Now

// PushSource is a MetricsSource that stores pushed metrics until the next scrape
// (instead of scraping from a remote location).
type PushSource interface {
	MetricsSource

	// PushMetrics stores a batch of metrics to be retrived by the next call to ScrapeMetrics.
	// The metrics should already be prefixed/namespaced by sourceName as appropriate.  It
	// returns an error if the push would have violated the metric name limit
	// (based on the set of names added).  It is expected that the set of names passed
	// matches the set of names present in the input batch, but this is not enforced
	// by the method.
	PushMetrics(batch *DataBatch, sourceName string, names map[string]struct{}) error
}

// pushProvider is a MetricsSourceProvider which returns both some sources from a base
// MetricsSourceProvider, as well as a PushSource
type pushProvider struct {
	baseProvider MetricsSourceProvider
	pushSource   PushSource
}

func (p *pushProvider) GetMetricsSources() []MetricsSource {
	var sources []MetricsSource
	if p.baseProvider != nil {
		sources = p.baseProvider.GetMetricsSources()
	} else {
		sources = make([]MetricsSource, 0, 1)
	}

	sources = append(sources, p.pushSource)

	return sources
}

// NewPushProvider returns a new MetricsSourceProvider which provides sources from the given
// MetricsSourceProvider (if any), as well as providing a PushSource.  If metricsLimit is non-zero,
// source will be limitted to that many metric names per scrape.
func NewPushProvider(baseProvider MetricsSourceProvider, metricsLimit int) (MetricsSourceProvider, PushSource, error) {
	pushSource := &memoryPushSource{
		dataBatches:  make(map[string]DataBatch),
		metricNames:  make(map[string]map[string]struct{}),
		metricsLimit: metricsLimit,
	}

	provider := &pushProvider{
		baseProvider: baseProvider,
		pushSource:   pushSource,
	}

	return provider, pushSource, nil
}

type memoryPushSource struct {
	dataBatches  map[string]DataBatch
	metricNames  map[string]map[string]struct{}
	metricsLimit int
	sync.Mutex
}

func (src *memoryPushSource) String() string {
	return "push"
}

func (src *memoryPushSource) Name() string {
	return src.String()
}

func (src *memoryPushSource) IsCanonical() bool {
	return false
}

func (src *memoryPushSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	src.Lock()
	defer src.Unlock()

	resBatch := &DataBatch{
		Timestamp:  nowFunc(),
		MetricSets: map[string]*MetricSet{},
	}

	for sourceName, batch := range src.dataBatches {
		// clear/free the dataBatch for when we "shorten" the slice later
		if (!start.IsZero() && batch.Timestamp.Before(start)) || (!end.IsZero() && batch.Timestamp.After(end)) {
			glog.V(2).Infof("Data batch from %q with invalid timestamp %s (not in [%s, %s])", "", batch.Timestamp, start, end)
			continue
		}

		glog.V(5).Infof("Processing pushed batch from source %q: %#v", sourceName, batch)

		mergeMetricSets(resBatch.MetricSets, batch.MetricSets, false)
		delete(src.dataBatches, sourceName)
		delete(src.metricNames, sourceName)
	}

	glog.V(5).Infof("Scraped batch from push source: %#v", resBatch)

	return resBatch
}

func mergeTyped(destValues map[string]MetricValue, newValues map[string]MetricValue, _ bool) {
	for valKey, newVal := range newValues {
		// for gauges, we should always average multiple pushes in a single
		// scrape period so that we don't lose values
		if newVal.MetricType == MetricGauge {
			if oldVal, oldValPresent := destValues[valKey]; oldValPresent {
				// we use the unused value to keep track of the count
				// (this is fine because the exporter just uses the correct value according to the value type)
				// This is a *bit* hacky, but it's easier than keeping track of the averages elsewhere and then
				// having to do an extra pass at the end.

				// TODO: can we just assume gauges are going to be float values?
				if oldVal.ValueType == ValueInt64 {
					if oldVal.FloatValue == 0 {
						// start a new running average
						newVal.FloatValue = 2.0
						newVal.IntValue = (oldVal.IntValue + newVal.IntValue) / 2
					} else {
						newVal.IntValue = (oldVal.IntValue*int64(oldVal.FloatValue) + newVal.IntValue) / int64(oldVal.FloatValue+1)
						newVal.FloatValue = oldVal.FloatValue + 1
					}
				} else {
					if oldVal.IntValue == 0 {
						// start a new running average
						newVal.IntValue = 2
						newVal.FloatValue = (oldVal.FloatValue + newVal.FloatValue) / 2
					} else {
						// continue the existing running average
						newVal.FloatValue = (oldVal.FloatValue*float32(oldVal.IntValue) + newVal.FloatValue) / float32(oldVal.IntValue+1)
						newVal.IntValue = oldVal.IntValue + 1

					}
				}
			}
		}

		destValues[valKey] = newVal
	}
}

func mergeMetricSets(dest map[string]*MetricSet, src map[string]*MetricSet, overwrite bool) {
	for setKey, newSet := range src {
		presentSet, wasPresent := dest[setKey]
		if !wasPresent {
			dest[setKey] = newSet
			continue
		}

		// set the scrape time to be the newest time
		if presentSet.ScrapeTime.Before(newSet.ScrapeTime) {
			presentSet.ScrapeTime = newSet.ScrapeTime
		}

		// do the rest of the merging
		if overwrite {
			util.MergeMetricSetCustom(presentSet, newSet, overwrite, mergeTyped)
		} else {
			util.MergeMetricSet(presentSet, newSet, overwrite)
		}
	}
}

// PushMetrics stores a batch of metrics to be retrived by the next call to ScrapeMetrics
func (src *memoryPushSource) PushMetrics(batch *DataBatch, sourceName string, newMetricNames map[string]struct{}) error {
	src.Lock()
	defer src.Unlock()

	// make sure this wouldn't violate the limit
	if src.metricsLimit > 0 {
		sourceMetricNames, ok := src.metricNames[sourceName]
		if !ok {
			sourceMetricNames := make(map[string]struct{})
			src.metricNames[sourceName] = sourceMetricNames
		}

		metricsCount := len(src.metricNames[sourceName])
		for name := range newMetricNames {
			if _, ok := sourceMetricNames[name]; !ok {
				metricsCount++
			}
		}

		if metricsCount > src.metricsLimit {
			return fmt.Errorf("only %v distinctly-named metrics may be added per batch, but %v would have been pushed", src.metricsLimit, metricsCount)
		}

		// actually record the metrics for future use
		for name := range newMetricNames {
			sourceMetricNames[name] = struct{}{}
		}
	}

	// if an entry already exists, merge it with the existing set
	// (overwriting duplicate entries), since we could have multiple
	// producers with the same name (e.g. the same daemon on each node)
	if oldBatch, ok := src.dataBatches[sourceName]; ok {
		mergeMetricSets(oldBatch.MetricSets, batch.MetricSets, true)
		return nil
	}

	src.dataBatches[sourceName] = *batch

	return nil
}
