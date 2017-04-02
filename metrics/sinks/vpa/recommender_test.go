/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vpa

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/metrics/core"
	"reflect"
	"testing"
	"time"
)

type fakeJSONClient struct {
	objectsSent []interface{}
}

func (client *fakeJSONClient) SendJSON(object interface{}) ([]byte, error) {
	client.objectsSent = append(client.objectsSent, object)
	return []byte{}, nil
}

func createFakeRecommenderSink() (core.DataSink, *fakeJSONClient) {
	fakeJSONClient := &fakeJSONClient{objectsSent: make([]interface{}, 0, 10)}
	return &recommenderSink{recommenderClient: fakeJSONClient}, fakeJSONClient
}

func createContainerMetricSet(containerName, containerImage, podId string, cpuUsageRate, memoryUsage int64) *core.MetricSet {
	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour)

	containerLabels := map[string]string{
		core.LabelContainerName.Key:      containerName,
		core.LabelContainerBaseImage.Key: containerImage,
		core.LabelPodId.Key:              podId,
		core.LabelNamespaceName.Key:      "default-namespace",
	}

	metricValues := createMetricValues(cpuUsageRate, memoryUsage, 1000, 2000)

	return &core.MetricSet{
		CreateTime:   hourAgo,
		ScrapeTime:   now,
		Labels:       containerLabels,
		MetricValues: metricValues,
	}
}

func createPodMetricSet(podId string, cpuUsageRate, memoryUsage int64) *core.MetricSet {
	metricSet := createContainerMetricSet("", "", podId, cpuUsageRate, memoryUsage)

	delete(metricSet.Labels, core.LabelContainerName.Key)
	delete(metricSet.Labels, core.LabelContainerBaseImage.Key)

	return metricSet
}

func createMetricValues(cpuUsageRate, memoryUsage, cpuRequest, memoryRequest int64) map[string]core.MetricValue {
	return map[string]core.MetricValue{
		core.MetricCpuUsageRate.Name: {
			MetricType: core.MetricCpuUsageRate.Type,
			ValueType:  core.MetricCpuUsageRate.ValueType,
			IntValue:   cpuUsageRate,
		},
		core.MetricMemoryUsage.Name: {
			MetricType: core.MetricMemoryUsage.Type,
			ValueType:  core.MetricMemoryUsage.ValueType,
			IntValue:   memoryUsage,
		},
		core.MetricMemoryRequest.Name: {
			MetricType: core.MetricMemoryRequest.Type,
			ValueType:  core.MetricMemoryRequest.ValueType,
			IntValue:   memoryRequest,
		},
		core.MetricCpuRequest.Name: {
			MetricType: core.MetricCpuRequest.Type,
			ValueType:  core.MetricCpuRequest.ValueType,
			IntValue:   cpuRequest,
		},
	}
}

func TestEmptyExportData(t *testing.T) {
	sink, jsonClient := createFakeRecommenderSink()
	batch := core.DataBatch{
		Timestamp:  time.Now(),
		MetricSets: map[string]*core.MetricSet{},
	}

	sink.ExportData(&batch)

	assert.Empty(t, jsonClient.objectsSent)
}

func TestExportData(t *testing.T) {
	sink, jsonClient := createFakeRecommenderSink()

	containerSet1 := createContainerMetricSet("container1", "gcr.io/google_containers/image1", "123", 123, 456)
	containerSet2 := createContainerMetricSet("container2", "gcr.io/google_containers/image2", "234", 234, 567)
	containerSet3 := createContainerMetricSet("container3", "gcr.io/google_containers/image3", "345", 345, 678)

	podSet1 := createPodMetricSet("pod1", 1234, 5678)
	podSet2 := createPodMetricSet("pod2", 2345, 6789)

	metricSets := map[string]*core.MetricSet{
		"container1": containerSet1,
		"container2": containerSet2,
		"container3": containerSet3,
		"pod1":       podSet1,
		"pod2":       podSet2,
	}

	batch := core.DataBatch{
		Timestamp:  time.Now(),
		MetricSets: metricSets,
	}

	sink.ExportData(&batch)

	assert.Equal(t, 1, len(jsonClient.objectsSent))

	actualObjectType := reflect.TypeOf((*utilizationSnapshotSet)(nil)).Elem()
	sentObjectType := reflect.TypeOf(jsonClient.objectsSent[0]).Elem()

	assert.Equal(t, actualObjectType, sentObjectType, "Objects sent should be of 'containerUtilizationSnapshot' type")

	set := jsonClient.objectsSent[0].(*utilizationSnapshotSet)
	assert.Equal(t, 3, len(set.Containers))
}

const (
	containerName  = "container"
	containerImage = "gcr.io/google_containers/image1"
	podId          = "123"
	cpuUsageRate   = int64(123)
	memoryUsage    = int64(456)
)

func TestContainerUtilizationSnapshotCreation(t *testing.T) {
	metricSet := createContainerMetricSet(containerName, containerImage, podId, cpuUsageRate, memoryUsage)
	snapshot, err := newContainerUtilizationSnapshot(metricSet)

	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, containerName, snapshot.ContainerName)
	assert.Equal(t, containerImage, snapshot.ContainerImage)
	assert.Equal(t, podId, snapshot.PodId)
	assert.Equal(t, cpuUsageRate, snapshot.CpuUsageRate)
	assert.Equal(t, memoryUsage, snapshot.MemoryUsage)
}

func TestContainerUtilizationSnapshotCreationError(t *testing.T) {
	metricSet := createContainerMetricSet("", containerImage, podId, cpuUsageRate, memoryUsage)
	_, err := newContainerUtilizationSnapshot(metricSet)
	assert.Error(t, err)

	metricSet = createContainerMetricSet(containerName, "", podId, cpuUsageRate, memoryUsage)
	_, err = newContainerUtilizationSnapshot(metricSet)
	assert.Error(t, err)

	metricSet = createContainerMetricSet(containerName, containerImage, "", cpuUsageRate, memoryUsage)
	_, err = newContainerUtilizationSnapshot(metricSet)
	assert.Error(t, err)

	metricSet = createContainerMetricSet(containerName, containerImage, podId, cpuUsageRate, memoryUsage)
	metricSet.Labels = nil
	_, err = newContainerUtilizationSnapshot(metricSet)
	assert.Error(t, err)
}
