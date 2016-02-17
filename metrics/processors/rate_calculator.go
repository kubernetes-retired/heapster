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
	"k8s.io/heapster/metrics/core"
)

type RateCalculator struct {
	rateMetricsMapping map[string]core.Metric
	previousBatch      *core.DataBatch
}

func (this *RateCalculator) Name() string {
	return "rate calculator"
}

func (this *RateCalculator) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	if this.previousBatch == nil {
		this.previousBatch = batch
		return batch, nil
	}

	for key, newMs := range batch.MetricSets {

		if oldMs, found := this.previousBatch.MetricSets[key]; found {
			if newMs.ScrapeTime.UnixNano()-oldMs.ScrapeTime.UnixNano() <= 0 {
				continue
			}

			for metricName, targetMetric := range this.rateMetricsMapping {
				metricValNew, foundNew := newMs.MetricValues[metricName]
				metricValOld, foundOld := oldMs.MetricValues[metricName]
				if foundNew && foundOld {
					if metricName == core.MetricCpuUsage.MetricDescriptor.Name {
						// cpu/usage values are in nanoseconds; we want to have it in millicores (that's why constant 1000 is here).
						newVal := 1000 * (metricValNew.IntValue - metricValOld.IntValue) /
							(newMs.ScrapeTime.UnixNano() - oldMs.ScrapeTime.UnixNano())

						newMs.MetricValues[targetMetric.MetricDescriptor.Name] = core.MetricValue{
							ValueType:  core.ValueInt64,
							MetricType: core.MetricGauge,
							IntValue:   newVal,
						}

					} else if targetMetric.MetricDescriptor.ValueType == core.ValueFloat {
						newVal := 1e9 * float32(metricValNew.IntValue-metricValOld.IntValue) /
							float32(newMs.ScrapeTime.UnixNano()-oldMs.ScrapeTime.UnixNano())

						newMs.MetricValues[targetMetric.MetricDescriptor.Name] = core.MetricValue{
							ValueType:  core.ValueFloat,
							MetricType: core.MetricGauge,
							FloatValue: newVal,
						}
					}
				}
			}
		}
	}
	this.previousBatch = batch
	return batch, nil
}

func NewRateCalculator(metrics map[string]core.Metric) *RateCalculator {
	return &RateCalculator{
		rateMetricsMapping: metrics,
	}
}
