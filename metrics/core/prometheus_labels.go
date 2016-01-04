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

//define the summary and histogram label for prometheus
package core

import (
	"github.com/prometheus/client_golang/prometheus"
)

const PrometheusPath = "/debug/prometheus"
const normDomain float64 = 200
const normMean float64 = 10

var (

	//time of the last metrics scrape
	LastMetricsTimeStamp = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "timeStamp_of_last_metricsScrape",
			Help: "TimeStamp of the last metrics scrape",
		},
		[]string{"timestamp"},
	)

	//histogram of Kubelet/source response times
	SourceResponseTimesHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "kubelet_source_response_count",
		Help:    "total times of Kubelet/Source response",
		Buckets: prometheus.LinearBuckets(normMean-5*normDomain, 5*normDomain, 20),
	})

	//time spent in scrape (single run, not cumulative)
	ScrapeDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "scrape_durations_microseconds",
			Help: "time spent in scrape",
		},
		[]string{"duration"},
	)

	//time spent in processors (single run, not cumulative)
	ProcessorDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "processor_durations_microseconds",
			Help: "time spent in processor",
		},
		[]string{"duration", "processor"},
	)

	//time spent in exporting data to sinks (single run, not cumulative)
	ExportingDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "exporting_durations_microseconds",
			Help: "time spent in exporting",
		},
		[]string{"duration"},
	)

	//number of scraped nodes
	ScrapedNodesCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "scraped_nodes_count",
			Help: "number of scraped nodes",
		},
	)

	//number of scraped containers
	ScrapedContainersCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "scraped_containers_count",
			Help: "number of scraped containers",
		},
	)

	//number of http model api requests
	ModelApiRequestCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "model_api_request_count",
			Help: "number of http model api requests",
		},
	)

	//number of other http requests
	OtherApiRequestCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "other_api_request_count",
			Help: "number of other model api requests",
		},
	)

	PrometheusHandler = prometheus.Handler()
)

func init() {
	prometheus.MustRegister(LastMetricsTimeStamp)
	prometheus.MustRegister(SourceResponseTimesHistogram)
	prometheus.MustRegister(ProcessorDurations)
	prometheus.MustRegister(ExportingDurations)
	prometheus.MustRegister(ScrapedNodesCount)
	prometheus.MustRegister(ScrapedContainersCount)
	prometheus.MustRegister(ModelApiRequestCount)
	prometheus.MustRegister(OtherApiRequestCount)
}
