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

package schema

import (
	"errors"
	"time"

	"github.com/GoogleCloudPlatform/heapster/api/schema/info"
	"github.com/GoogleCloudPlatform/heapster/sinks/cache"
)

func maxTimestamp(first time.Time, second time.Time) time.Time {
	if first.After(second) {
		return first
	} else {
		return second
	}
}

func updateInfoType(info *info.InfoType, ce *cache.ContainerElement) (time.Time, error) {
	// Updates the metrics of an InfoType struct from a ContainerElement struct
	//  Returns the max timestamp in the resulting timeseries slice

	var latest_time time.Time
	var err error

	if ce == nil {
		err = errors.New("Cannot update InfoType from Nil ContainerElement")
		return latest_time, err
	} else if info == nil {
		err = errors.New("Cannot update Nil InfoType")
		return latest_time, err
	}

	new_metrics, err := parseMetrics(ce.Metrics)

	// Update Metrics
	for key, metricSlice := range new_metrics {
		if val, ok := info.Metrics[key]; ok {
			// Metric already exists in info, merge timeseries slices
			info.Metrics[key] = append(val, metricSlice...)
		} else {
			// New Metric to add to info
			info.Metrics[key] = metricSlice
		}

		if val, ok := info.Metrics[key]; ok {
			// Calculate new latest timestamp
			for _, metric := range val {
				latest_time = maxTimestamp(latest_time, metric.Timestamp)
			}
		}
	}
	// TODO: manage length of historical data

	return latest_time, err
}

func newInfoType(metrics map[string][]*info.MetricTimeseries, labels map[string]string) info.InfoType {
	// InfoType Constructor
	if metrics == nil {
		metrics = make(map[string][]*info.MetricTimeseries)
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	return info.InfoType{
		Metrics: metrics,
		Labels:  labels,
	}
}

func addMetricToMap(metric string, timestamp time.Time, value uint64, dict_ref *map[string][]*info.MetricTimeseries) error {
	// Adds a metric to a map of MetricTimeseries
	dict := *dict_ref
	if val, ok := dict[metric]; ok {
		dict[metric] = append(val, &info.MetricTimeseries{
			Timestamp: timestamp,
			Value:     value,
		})
	} else {
		new_timeseries := &info.MetricTimeseries{
			Timestamp: timestamp,
			Value:     value,
		}
		dict[metric] = []*info.MetricTimeseries{new_timeseries}
	}
	return nil
}

func parseMetrics(cmes []*cache.ContainerMetricElement) (map[string][]*info.MetricTimeseries, error) {
	// Generates a map of MetricTimeseries slices from a slice of ContainerMetricElements

	var err error

	metrics := make(map[string][]*info.MetricTimeseries)

	if cmes == nil || len(cmes) == 0 {
		err = errors.New("Cannot parse empty slice of containerMetricElement")
		return nil, err
	}

	for _, cme := range cmes {
		if cme == nil {
			err = errors.New("Nil CME found in slice")
			return nil, err
		}
		timestamp := cme.Stats.Timestamp
		if cme.Spec.HasCpu {
			// Add CPU limit
			cpu_limit := cme.Spec.Cpu.Limit
			addMetricToMap("cpu/limit", timestamp, cpu_limit, &metrics)

			// Add CPU metric
			cpu_usage := cme.Stats.Cpu.Usage.Total
			addMetricToMap("cpu/usage", timestamp, cpu_usage, &metrics)
		}

		if cme.Spec.HasMemory {
			// Add Memory Limit metric
			mem_limit := cme.Spec.Memory.Limit

			// TODO: -1 values from cache?
			addMetricToMap("memory/limit", timestamp, mem_limit, &metrics)

			// Add Memory Usage metric
			mem_usage := cme.Stats.Memory.Usage
			addMetricToMap("memory/usage", timestamp, mem_usage, &metrics)

			// Add Memory Working Set metric
			mem_working := cme.Stats.Memory.WorkingSet
			addMetricToMap("memory/working", timestamp, mem_working, &metrics)
		}
		if cme.Spec.HasFilesystem {
			for _, fsstat := range cme.Stats.Filesystem {
				dev := fsstat.Device

				// Add FS limit, if applicable
				fs_limit := fsstat.Limit
				addMetricToMap("fs/limit"+dev, timestamp, fs_limit, &metrics)

				// Add FS metric
				fs_usage := fsstat.Usage
				addMetricToMap("fs/usage"+dev, timestamp, fs_usage, &metrics)
			}
		}
	}
	return metrics, err
}

func addContainerToMap(container_name string, target_map *map[string]*info.ContainerInfo) *info.ContainerInfo {
	//	Creates or finds a ContainerInfo element under a *map[string]*ContainerInfo
	//	Assumes Cluster lock is already taken
	var container_ptr *info.ContainerInfo

	dict := *target_map
	if val, ok := dict[container_name]; ok {
		// A container already exists under that name, return pointer
		container_ptr = val
	} else {
		container_ptr = &info.ContainerInfo{
			InfoType: newInfoType(nil, nil),
		}
		dict[container_name] = container_ptr
	}
	return container_ptr
}
