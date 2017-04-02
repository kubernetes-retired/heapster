// Copyright 2017 Google Inc. All Rights Reserved.
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

package vpa

import (
	"github.com/pkg/errors"
	"k8s.io/heapster/metrics/core"
	"time"
)

type utilizationSnapshotSet struct {
	Containers []*containerUtilizationSnapshot `json:"containers"`
}

func (set *utilizationSnapshotSet) addContainer(utilizationSnapshot *containerUtilizationSnapshot) {
	set.Containers = append(set.Containers, utilizationSnapshot)
}

type containerUtilizationSnapshot struct {
	CreateTime     time.Time `json:"createTime"`
	ScrapTime      time.Time `json:"scrapTime"`
	ContainerName  string    `json:"containerName"`
	ContainerImage string    `json:"containerImage"`
	PodId          string    `json:"podId"`
	// Amount of requested memory.
	// Units: Bytes.
	MemoryRequest int64 `json:"memoryRequest"`
	// Current, total memory usage
	// Units: Bytes.
	MemoryUsage int64 `json:"memoryUsage"`
	// Guaranteed amount of CPU
	// Units: Millicores.
	CpuRequest int64 `json:"cpuRequest"`
	// Avg CPU usage since last scrap time
	// Units: Millicores.
	CpuUsageRate int64 `json:"cpuUsageRate"`
}

func newContainerUtilizationSnapshot(metricSet *core.MetricSet) (*containerUtilizationSnapshot, error) {
	err := validateMetricSet(metricSet)
	if err != nil {
		return nil, err
	}

	snapshot := containerUtilizationSnapshot{
		CreateTime:     metricSet.CreateTime,
		ScrapTime:      metricSet.ScrapeTime,
		ContainerName:  metricSet.Labels[core.LabelContainerName.Key],
		ContainerImage: metricSet.Labels[core.LabelContainerBaseImage.Key],
		PodId:          metricSet.Labels[core.LabelPodId.Key],
		MemoryRequest:  metricSet.MetricValues[core.MetricMemoryRequest.Name].IntValue,
		MemoryUsage:    metricSet.MetricValues[core.MetricMemoryUsage.Name].IntValue,
		CpuRequest:     metricSet.MetricValues[core.MetricCpuRequest.Name].IntValue,
		CpuUsageRate:   metricSet.MetricValues[core.MetricCpuUsageRate.Name].IntValue,
	}
	return &snapshot, nil
}

func validateMetricSet(metricSet *core.MetricSet) error {
	if metricSet.Labels == nil || metricSet.MetricValues == nil {
		return errors.New("MetricSet needs both 'Labels' and 'MetricValues' not null")
	}
	if metricSet.Labels[core.LabelContainerName.Key] == "" ||
		metricSet.Labels[core.LabelContainerBaseImage.Key] == "" ||
		metricSet.Labels[core.LabelPodId.Key] == "" {

		return errors.Errorf("MetricSet is missing one of the labels: %s, %s, %s", core.LabelContainerName.Key,
			core.LabelContainerBaseImage.Key, core.LabelPodId.Key)
	}

	return nil
}
