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

package summary

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/golang/glog"
	. "k8s.io/heapster/metrics/core"
)

var (
	// Ignore these metrics.
	ignoreMetrics = map[string]bool{
		MetricUptime.Name: true, // Checked via creation time
	}

	// Ignore these metrics on pod metric sets.
	ignorePodMetrics = map[string]bool{
		// The following are set from the POD infra container using the old kubelet metrics,
		// but not set for pods using the summary_api.
		"cpu/usage":                true,
		"memory/working_set":       true,
		"memory/usage":             true,
		"memory/page_faults":       true,
		"memory/major_page_faults": true,
	}

	// Ignore these labels.
	ignoreLabels = map[string]bool{
		LabelContainerBaseImage.Key: true, // Set in pod_based_enricher.go
	}
)

type CompareSource struct {
	summary *summaryMetricsSource
	kubelet MetricsSource
}

func NewCompareSource(summary *summaryMetricsSource) MetricsSource {
	return &CompareSource{
		summary: summary,
		kubelet: summary.fallback,
	}
}

func (this *CompareSource) Name() string   { return this.String() }
func (this *CompareSource) String() string { return "__compare__:" + this.summary.String() }

func (this *CompareSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	var (
		summaryMetrics *DataBatch
		kubeletMetrics *DataBatch
	)
	summaryMetrics = this.summary.ScrapeMetrics(start, end)
	kubeletMetrics = this.kubelet.ScrapeMetrics(start, summaryMetrics.Timestamp)

	if this.summary.useFallback {
		this.errorf("summary used fallback")
		return nil
	}

	this.validateMetrics(summaryMetrics, kubeletMetrics)
	return summaryMetrics
}

var (
	podKeyRe = regexp.MustCompile("namespace:[^:/]+/pod:[^:/]+")
)

func (this *CompareSource) validateMetrics(summary, kubelet *DataBatch) {
	if summary == nil {
		this.errorf("summary metrics is nil")
		return
	}
	if kubelet == nil {
		this.errorf("kubelet metrics is nil")
		return
	}
	valid := true
	if !fuzzyTimeEqual(summary.Timestamp, kubelet.Timestamp, 5*time.Second) {
		valid = this.mismatch("timestamp", summary.Timestamp, kubelet.Timestamp)
	}

	for name, kubeletSet := range kubelet.MetricSets {
		summarySet, found := summary.MetricSets[name]
		if !found {
			valid = this.errorf("Summary missing metric set %q", name)
			continue
		}
		if !fuzzyTimeEqual(summarySet.CreateTime, kubeletSet.CreateTime, time.Second) {
			valid = this.mismatch("creation time", summarySet.CreateTime, kubeletSet.CreateTime)
			continue
		}
		if !fuzzyTimeEqual(summarySet.ScrapeTime, kubeletSet.ScrapeTime, time.Second) {
			valid = this.mismatch("scrape time", summarySet.ScrapeTime, kubeletSet.ScrapeTime)
			continue
		}

		isPod := podKeyRe.MatchString(name)
		for metric, kubeVal := range kubeletSet.MetricValues {
			if isPod && ignorePodMetrics[metric] {
				continue
			}
			sumVal, found := summarySet.MetricValues[metric]
			if !found {
				valid = this.errorf("summary set %q missing metric %q", name, metric)
				continue
			}
			var mismatch bool
			switch metric {
			case MetricUptime.Name:
				// Uptime depends on exact scrape time, so perform a fuzzy comparison.
				diff := sumVal.IntValue - kubeVal.IntValue
				const epsilon = int64(time.Second / time.Millisecond)
				mismatch = diff < -epsilon || epsilon < diff
			default:
				mismatch = sumVal != kubeVal
			}
			if mismatch {
				valid = this.mismatch(name+" metric "+metric, sumVal, kubeVal)
			}
		}

		for label, kv := range kubeletSet.Labels {
			if ignoreLabels[label] {
				continue
			}
			sv, found := summarySet.Labels[label]
			if !found {
				valid = this.errorf("summary set %q missing label %q", name, label)
			}
			if kv != sv {
				valid = this.mismatch(name+" label "+label, sv, kv)
			}
		}

		// TODO: validate labeled metrics.
	}

	if valid {
		glog.Infof("!!!! metric set valid!")
	} else {
		glog.Fatalf("!!!! metric set invalid")
	}
}

func (this *CompareSource) errorf(format string, args ...interface{}) bool {
	const tag = ">>> "
	fmt.Printf(tag+format+"\n", args...)
	//	glog.Errorf(tag+format, args...)
	return false
}

func (this *CompareSource) mismatch(label string, summary, kubelet interface{}) bool {
	return this.errorf(label+" mismatch: summary = %v kubelet = %v", summary, kubelet)
}

func fuzzyTimeEqual(a, b time.Time, epsilon time.Duration) bool {
	diff := a.Sub(b)
	return -epsilon <= diff && diff <= epsilon
}

type CompareProvider struct {
	summary *summaryProvider
}

func NewCompareProvider(uri *url.URL) (MetricsSourceProvider, error) {
	summary, err := NewSummaryProvider(uri)
	if err != nil {
		return nil, err
	}
	return &CompareProvider{
		summary: summary.(*summaryProvider),
	}, nil
}

func (this *CompareProvider) GetMetricsSources() []MetricsSource {
	summarySources := this.summary.GetMetricsSources()
	sources := make([]MetricsSource, len(summarySources))
	for i, source := range summarySources {
		summary := source.(*summaryMetricsSource)
		if summary.useFallback {
			glog.Errorf("**** New source %q using fallback!!!!", sources[i].Name())
		}
		sources[i] = NewCompareSource(summary)
	}
	return sources
}
