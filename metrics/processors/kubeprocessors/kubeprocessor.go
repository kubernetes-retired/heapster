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

package kubeprocessors

import (
	"net/url"

	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
)

func GetKubeProcessors(url *url.URL) ([]core.DataProcessor, error) {
	// data processors
	metricsToAggregate := []string{
		core.MetricCpuUsageRate.Name,
		core.MetricMemoryUsage.Name,
		core.MetricCpuRequest.Name,
		core.MetricCpuLimit.Name,
		core.MetricMemoryRequest.Name,
		core.MetricMemoryLimit.Name,
	}

	metricsToAggregateForNode := []string{
		core.MetricCpuRequest.Name,
		core.MetricCpuLimit.Name,
		core.MetricMemoryRequest.Name,
		core.MetricMemoryLimit.Name,
	}

	dataProcessors := []core.DataProcessor{
		// Convert cumulaties to rate
		NewRateCalculator(core.RateMetricsMapping),
	}

	podBasedEnricher, err := NewPodBasedEnricher(url)
	if err != nil {
		glog.Fatalf("Failed to create PodBasedEnricher: %v", err)
		return nil, err
	}
	dataProcessors = append(dataProcessors, podBasedEnricher)

	namespaceBasedEnricher, err := NewNamespaceBasedEnricher(url)
	if err != nil {
		glog.Fatalf("Failed to create NamespaceBasedEnricher: %v", err)
		return nil, err
	}
	dataProcessors = append(dataProcessors, namespaceBasedEnricher)

	// then aggregators
	dataProcessors = append(dataProcessors,
		NewPodAggregator(),
		&NamespaceAggregator{
			MetricsToAggregate: metricsToAggregate,
		},
		&NodeAggregator{
			MetricsToAggregate: metricsToAggregateForNode,
		},
		&ClusterAggregator{
			MetricsToAggregate: metricsToAggregate,
		})

	nodeAutoscalingEnricher, err := NewNodeAutoscalingEnricher(url)
	if err != nil {
		glog.Fatalf("Failed to create NodeAutoscalingEnricher: %v", err)
		return nil, err
	}
	dataProcessors = append(dataProcessors, nodeAutoscalingEnricher)
	return dataProcessors, nil
}
