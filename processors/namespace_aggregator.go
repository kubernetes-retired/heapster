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

package processors

import (
	"fmt"

	"k8s.io/heapster/core"
)

type NamespaceAggregator struct {
	MetricsToAggregate []string
}

func (this *NamespaceAggregator) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	result := core.DataBatch{
		Timestamp:  batch.Timestamp,
		MetricSets: make(map[string]*core.MetricSet),
	}

	for key, metricSet := range batch.MetricSets {
		result.MetricSets[key] = metricSet
		if metricSetType, found := metricSet.Labels[core.LabelMetricSetType.Key]; found && metricSetType == core.MetricSetTypePod {
			// Aggregating pods
			if namespaceName, found := metricSet.Labels[core.LabelNamespaceName.Key]; found {
				namespaceKey := core.NamespaceKey(namespaceName)
				namespace, found := result.MetricSets[namespaceKey]
				if !found {
					namespace = namespaceMetricSet(namespaceName)
					result.MetricSets[namespaceKey] = namespace
				}

				for _, metricName := range this.MetricsToAggregate {
					metricValue, found := metricSet.MetricValues[metricName]
					if !found {
						continue
					}
					aggregatedValue, found := namespace.MetricValues[metricName]
					if found {
						if aggregatedValue.ValueType != metricValue.ValueType {
							return nil, fmt.Errorf("NamespaceAggregator: type not supported in %s", metricName)
						}

						if aggregatedValue.ValueType == core.ValueInt64 {
							aggregatedValue.IntValue += metricValue.IntValue
						} else if aggregatedValue.ValueType == core.ValueFloat {
							aggregatedValue.FloatValue += metricValue.FloatValue
						} else {
							return nil, fmt.Errorf("NamespaceAggregator: type not supported in %s", metricName)
						}
					} else {
						aggregatedValue = metricValue
					}
					namespace.MetricValues[metricName] = aggregatedValue
				}
			} else {
				return nil, fmt.Errorf("No namespace info in pod %s: %v", key, metricSet.Labels)
			}
		}
	}
	return &result, nil
}

func namespaceMetricSet(namespaceName string) *core.MetricSet {
	return &core.MetricSet{
		MetricValues: make(map[string]core.MetricValue),
		Labels: map[string]string{
			core.LabelMetricSetType.Key: core.MetricSetTypeNamespace,
			core.LabelNamespaceName.Key: namespaceName,
		},
	}
}
