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

package sources

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "k8s.io/heapster/core"
)

const (
	DefaultMetricsScrapeTimeout = 20 * time.Second
)

// A place from where the metrics should be scraped.
type MetricsSource interface {
	ScrapeMetrics(start, end time.Time) *DataBatch
}

// Kubelet-provided metrics for pod and system container.
type KubeletMetricsSource struct {
	host string
	port int
}

func (this *KubeletMetricsSource) String() string {
	return fmt.Sprintf("kubelet:%s:%d", this.host, this.port)
}

func (this *KubeletMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	var tmp DataBatch
	return &tmp
}

// Provider of list of sources to be scaped.
type MetricsSourceProvider interface {
	GetMetricsSources() []MetricsSource
}

type KubeletProvider struct {
}

func (this *KubeletProvider) GetMetricsSources() []MetricsSource {
	return []MetricsSource{}
}

func NewSourceManager(metricsSourceProvider MetricsSourceProvider, metricsScrapeTimeout time.Duration) (MetricsSource, error) {
	return &sourceManager{
		metricsSourceProvider: metricsSourceProvider,
		metricsScrapeTimeout:  metricsScrapeTimeout,
	}, nil
}

type sourceManager struct {
	metricsSourceProvider MetricsSourceProvider
	metricsScrapeTimeout  time.Duration
}

func (this *sourceManager) ScrapeMetrics(start, end time.Time) *DataBatch {
	sources := this.metricsSourceProvider.GetMetricsSources()

	responseChannel := make(chan *DataBatch)
	timeout := time.Now().Add(this.metricsScrapeTimeout)

	for _, source := range sources {
		go func(source MetricsSource, channel chan *DataBatch, start, end, timeout time.Time) {
			glog.Infof("Querying source: %s", source)
			metrics := source.ScrapeMetrics(start, end)
			now := time.Now()
			if !now.Before(timeout) {
				glog.Warningf("Failed to get %s response in time", source)
				return
			}
			timeForResponse := timeout.Sub(now)

			select {
			case channel <- metrics:
				// passed the response correctly.
				return
			case <-time.After(timeForResponse):
				glog.Warningf("Failed to send the response back %s", source)
				return
			}
		}(source, responseChannel, start, end, timeout)
	}
	response := DataBatch{
		Timestamp:  end,
		MetricSets: map[string]*MetricSet{},
	}

responseloop:
	for i := range sources {
		now := time.Now()
		if !now.Before(timeout) {
			glog.Warningf("Failed to get all responses in time (got %d/%d)", i, len(sources))
			break
		}

		select {
		case dataBatch := <-responseChannel:
			if dataBatch != nil {
				for key, value := range dataBatch.MetricSets {
					response.MetricSets[key] = value
				}
			}
		case <-time.After(timeout.Sub(now)):
			glog.Warningf("Failed to get all responses in time (got %d/%d)", i, len(sources))
			break responseloop
		}
	}

	return &response
}
