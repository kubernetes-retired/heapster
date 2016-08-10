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

	"k8s.io/heapster/metrics/core"

	"github.com/golang/glog"
)

// for testing
var nowFunc = time.Now

// PushSource is a MetricsSource that stores pushed metrics until the next scrape
// (instead of scraping from a remote location).
type PushSource interface {
	core.MetricsSource

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
	baseProvider core.MetricsSourceProvider
	pushSource   PushSource
}

func (p *pushProvider) GetMetricsSources() []core.MetricsSource {
	var sources []core.MetricsSource
	if p.baseProvider != nil {
		sources = p.baseProvider.GetMetricsSources()
	} else {
		sources = make([]core.MetricsSource, 0, 1)
	}

	sources = append(sources, p.pushSource)

	return sources
}

// NewPushProvider returns a new MetricsSourceProvider which provides sources from the given
// MetricsSourceProvider (if any), as well as providing a PushSource.  If metricsLimit is non-zero,
// source will be limitted to that many metric names per scrape.
func NewPushProvider(baseProvider core.MetricsSourceProvider, metricsLimit int) (core.MetricsSourceProvider, PushSource, error) {
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

func (src *memoryPushSource) ScrapeMetrics(start, end time.Time) *core.DataBatch {
	src.Lock()
	defer src.Unlock()

	resBatch := &DataBatch{
		Timestamp:  nowFunc(),
		MetricSets: map[string]*MetricSet{},
	}

	for sourceName, batch := range src.dataBatches {
		if (!start.IsZero() && batch.Timestamp.Before(start)) || (!end.IsZero() && batch.Timestamp.After(end)) {
			glog.V(2).Infof("Data batch from %q with invalid timestamp %s (not in [%s, %s])", "", batch.Timestamp, start, end)
			continue
		}

		glog.V(5).Infof("Processing pushed batch from source %q: %#v", sourceName, batch)

		batch.MergeInto(resBatch.MetricSets, false)

		// clear/free the dataBatch for when we "shorten" the slice later
		delete(src.dataBatches, sourceName)
		delete(src.metricNames, sourceName)
	}

	glog.V(5).Infof("Scraped batch from push source: %#v", resBatch)

	return resBatch.DataBatch()
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
		// MetricSets is a map, so we don't need a pointer
		batch.MergeInto(oldBatch.MetricSets, true)
		return nil
	}

	src.dataBatches[sourceName] = *batch

	return nil
}
