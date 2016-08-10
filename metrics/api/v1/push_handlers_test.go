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

package v1

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sources/push"
)

var testPrometheusTextMetrics = `
# This is a pod-level metric (it might be used for autoscaling)
# TYPE http_requests_per_minute gauge
http_requests_per_minute{namespace="webapp",pod="frontend-server-a-1"} 20
http_requests_per_minute{namespace="webapp",pod="frontend-server-a-2"} 5
http_requests_per_minute{namespace="webapp",pod="frontend-server-b-1"} 25

# This is a service-level metric, which will be stored as frontend_hits_total
# and restapi_hits_total (these might be used for auto-idling)
# TYPE frontend_hits_total counter
frontend_hits_total{namespace="webapp"} 5000
# TYPE restapi_hits_total counter
restapi_hits_total{namespace="webapp"} 6000

# this is a labeled metric
restapi_hits{namespace="webapp",port="80",endpoint="/cheeses/pepper-jack"} 2000
restapi_hits{namespace="webapp",port="443",endpoint="/cheeses/cheddar"} 4000
`

func makePodMetrics(ns, name string, metricName string, metricVal float32) *push.MetricSet {
	return &push.MetricSet{
		MetricValues: map[string]push.MetricValue{
			metricName: {
				MetricValue: core.MetricValue{
					FloatValue: metricVal,
					ValueType:  core.ValueFloat,
					MetricType: core.MetricGauge,
				},
			},
		},
		Labels: map[string]string{
			core.LabelNamespaceName.Key: ns,
			core.LabelPodName.Key:       name,
			core.LabelMetricSetType.Key: core.MetricSetTypePod,
		},
		LabeledMetrics: nil,
	}
}

func makeLabeledMetrics(metrics []push.LabeledMetric) map[push.LabeledMetricID]push.LabeledMetric {
	res := make(map[push.LabeledMetricID]push.LabeledMetric, len(metrics))
	for _, metric := range metrics {
		res[metric.MakeID()] = metric
	}

	return res
}

func TestPrometheusTextIngest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	nowTime := time.Now().Truncate(time.Millisecond)
	nowFunc = func() time.Time { return nowTime }

	expectedResultBatch := &push.DataBatch{
		Timestamp: nowTime,
		MetricSets: map[string]*push.MetricSet{
			core.PodKey("webapp", "frontend-server-a-1"): makePodMetrics("webapp", "frontend-server-a-1", "custom/routermetrics/http_requests_per_minute", 20.0),
			core.PodKey("webapp", "frontend-server-a-2"): makePodMetrics("webapp", "frontend-server-a-2", "custom/routermetrics/http_requests_per_minute", 5.0),
			core.PodKey("webapp", "frontend-server-b-1"): makePodMetrics("webapp", "frontend-server-b-1", "custom/routermetrics/http_requests_per_minute", 25.0),

			core.NamespaceKey("webapp"): {
				MetricValues: map[string]push.MetricValue{
					"custom/routermetrics/frontend_hits_total": {
						MetricValue: core.MetricValue{
							FloatValue: 5000,
							ValueType:  core.ValueFloat,
							MetricType: core.MetricCumulative,
						},
					},
					"custom/routermetrics/restapi_hits_total": {
						MetricValue: core.MetricValue{
							FloatValue: 6000,
							ValueType:  core.ValueFloat,
							MetricType: core.MetricCumulative,
						},
					},
				},
				Labels: map[string]string{core.LabelNamespaceName.Key: "webapp", core.LabelMetricSetType.Key: core.MetricSetTypeNamespace},
				LabeledMetrics: makeLabeledMetrics([]push.LabeledMetric{
					{
						Name:   "custom/routermetrics/restapi_hits",
						Labels: []push.LabelPair{{"endpoint", "/cheeses/pepper-jack"}, {"port", "80"}},
						MetricValue: push.MetricValue{
							MetricValue: core.MetricValue{
								FloatValue: 2000,
								ValueType:  core.ValueFloat,
								MetricType: core.MetricGauge,
							},
						},
					},
					{
						Name:   "custom/routermetrics/restapi_hits",
						Labels: []push.LabelPair{{"endpoint", "/cheeses/cheddar"}, {"port", "443"}},
						MetricValue: push.MetricValue{
							MetricValue: core.MetricValue{
								FloatValue: 4000,
								ValueType:  core.ValueFloat,
								MetricType: core.MetricGauge,
							},
						},
					},
				}),
			},
		},
	}

	metricNames := make(map[string]struct{})
	batch, _, err := ingestPrometheusMetrics("routermetrics", metricNames, http.Header{
		"Content-Type": []string{"text/plain; version=0.0.4"},
	}, strings.NewReader(testPrometheusTextMetrics))
	require.NoError(err, "should have been able to process the metrics without error")
	assert.Equal(expectedResultBatch, batch, "ingested data batch should have been as expected")
}

func TestMetricNamesPopulation(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	nowTime := time.Now()
	nowFunc = func() time.Time { return nowTime }

	expectedMetricNames := map[string]struct{}{
		"custom/routermetrics/http_requests_per_minute": {},
		"custom/routermetrics/frontend_hits_total":      {},
		"custom/routermetrics/restapi_hits_total":       {},
		"custom/routermetrics/restapi_hits":             {},
	}

	metricNames := make(map[string]struct{})
	_, _, err := ingestPrometheusMetrics("routermetrics", metricNames, http.Header{
		"Content-Type": []string{"text/plain; version=0.0.4"},
	}, strings.NewReader(testPrometheusTextMetrics))
	require.NoError(err, "should have been able to process the metrics without error")
	assert.Equal(expectedMetricNames, metricNames, "should have recorded the metrics added")
}
