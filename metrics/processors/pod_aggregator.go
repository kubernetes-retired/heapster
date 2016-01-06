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

	"github.com/golang/glog"

	"k8s.io/heapster/metrics/core"
)

var LabelsToPopulate = []core.LabelDescriptor{
	core.LabelPodId,
	core.LabelPodName,
	core.LabelPodNamespace,
	core.LabelNamespaceName,
	core.LabelPodNamespaceUID,
	core.LabelHostname,
	core.LabelHostID,
	core.LabelCustomMetricName,
}

type PodAggregator struct {
}

func (this *PodAggregator) Name() string {
	return "pod_aggregator"
}

func (this *PodAggregator) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	result := core.DataBatch{
		Timestamp:  batch.Timestamp,
		MetricSets: make(map[string]*core.MetricSet),
	}
	for key, metricSet := range batch.MetricSets {
		result.MetricSets[key] = metricSet
		if metricSetType, found := metricSet.Labels[core.LabelMetricSetType.Key]; found && metricSetType == core.MetricSetTypePodContainer {
			// Aggregating containers
			podName, found := metricSet.Labels[core.LabelPodName.Key]
			ns, found2 := metricSet.Labels[core.LabelNamespaceName.Key]
			if found && found2 {
				podKey := core.PodKey(ns, podName)
				pod, found := result.MetricSets[podKey]
				if !found {
					pod = this.podMetricSet(metricSet.Labels)
					result.MetricSets[podKey] = pod
				}

				for metricName, metricValue := range metricSet.MetricValues {
					aggregatedValue, found := pod.MetricValues[metricName]
					if found {
						if aggregatedValue.ValueType != metricValue.ValueType {
							glog.Errorf("PodAggregator: inconsistent type in %s", metricName)
							continue
						}

						switch aggregatedValue.ValueType {
						case core.ValueInt64:
							aggregatedValue.IntValue += metricValue.IntValue
						case core.ValueFloat:
							aggregatedValue.FloatValue += metricValue.FloatValue
						default:
							return nil, fmt.Errorf("PodAggregator: type not supported in %s", metricName)
						}
					} else {
						aggregatedValue = metricValue
					}
					pod.MetricValues[metricName] = aggregatedValue
				}
			} else {
				glog.Errorf("No namespace and/or pod info in container %s: %v", key, metricSet.Labels)
				continue
			}
		}
	}

	return &result, nil
}

func (this *PodAggregator) podMetricSet(labels map[string]string) *core.MetricSet {
	newLabels := map[string]string{
		core.LabelMetricSetType.Key: core.MetricSetTypePod,
	}
	for _, l := range LabelsToPopulate {
		if val, ok := labels[l.Key]; ok {
			newLabels[l.Key] = val
		}
	}
	return &core.MetricSet{
		MetricValues: make(map[string]core.MetricValue),
		Labels:       newLabels,
	}
}
