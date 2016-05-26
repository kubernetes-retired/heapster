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

package apiserver

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/heapster/metrics/core"
	kube_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	kube_client "k8s.io/kubernetes/pkg/client/unversioned"
	"testing"
	"time"
)

func checkIntMetric(t *testing.T, metrics *core.MetricSet, key string, metric core.Metric, value int) {
	m, ok := metrics.MetricValues[metric.Name]
	if !assert.True(t, ok, "missing %q:%q", key, metric.Name) {
		return
	}
	assert.Equal(t, value, m.IntValue, "%q:%q", key, metric.Name)
}

type fakeStore struct {
	cache.Store
	objList []interface{}
}

func (f *fakeStore) Add(obj interface{}) error {
	f.objList = append(f.objList, obj)
	return nil
}

func (f *fakeStore) List() []interface{} {
	return f.objList
}

type fakeIndexer struct {
	cache.Indexer
	objList []interface{}
}

func (f *fakeIndexer) Add(obj interface{}) error {
	f.objList = append(f.objList, obj)
	return nil
}

func (f *fakeIndexer) List() []interface{} {
	return f.objList
}

func testingApiServerMetricsSource() *apiServerMetricsSource {
	dplLister := cache.StoreToDeploymentLister{}
	dplLister.Store = &fakeStore{}
	rsLister := cache.StoreToReplicaSetLister{}
	rsLister.Store = &fakeStore{}
	rcLister := &cache.StoreToReplicationControllerLister{}
	rcLister.Indexer = &fakeIndexer{}
	dsLister := cache.StoreToDaemonSetLister{}
	dsLister.Store = &fakeStore{}

	return &apiServerMetricsSource{
		kubeClient: &kube_client.Client{},
		dplLister:  dplLister,
		rsLister:   rsLister,
		rcLister:   rcLister,
		dsLister:   dsLister,
	}
}

func TestDecodeApiServerMetrics(t *testing.T) {
	// prepare source
	ms := testingApiServerMetricsSource()

	// Add dpl1
	dpl1 := &extensions.Deployment{}
	dpl1.Name = "deployment_test_1"
	dpl1.Spec.Replicas = 4
	dpl1.Status.Replicas = 4
	dpl1.Status.UpdatedReplicas = 4
	dpl1.Status.AvailableReplicas = 4
	dpl1.Status.UnavailableReplicas = 0
	ms.dplLister.Add(dpl1)
	// Add ReplicaSet
	rs1 := &extensions.ReplicaSet{}
	rs1.Name = "replicaset_test_1"
	rs1.Spec.Replicas = 3
	rs1.Status.Replicas = 3
	rs1.Status.FullyLabeledReplicas = 3
	ms.rsLister.Add(rs1)
	// Add ReplicationController
	rc1 := &kube_api.ReplicationController{}
	rc1.Name = "replicationcontroller_test_1"
	rc1.Spec.Replicas = 5
	rc1.Status.Replicas = 5
	rc1.Status.FullyLabeledReplicas = 5
	ms.rcLister.Add(rc1)
	// Add ds1
	ds1 := &extensions.DaemonSet{}
	ds1.Name = "daemonset_test_1"
	ds1.Status.CurrentNumberScheduled = 6
	ds1.Status.NumberMisscheduled = 0
	ds1.Status.DesiredNumberScheduled = 6
	ms.dsLister.Add(ds1)

	// Get metrics
	results := ms.ScrapeMetrics(time.Now(), time.Now())

	// Prepare expectations
	expectations := []struct {
		key        string
		metricType core.Metric
		value      int
		cpu        bool
		memory     bool
		network    bool
		fs         []string
	}{{
		key:        "deployment:deployment_test_1",
		metricType: core.MetricScheduledReplicas,
		value:      4,
	}, {
		key:        "deployment:deployment_test_1",
		metricType: core.MetricDesiredReplicas,
		value:      4,
	}, {
		key:        "deployment:deployment_test_1",
		metricType: core.MetricUpdatedReplicas,
		value:      4,
	}, {
		key:        "deployment:deployment_test_1",
		metricType: core.MetricAvailableReplicas,
		value:      4,
	}, {
		key:        "deployment:deployment_test_1",
		metricType: core.MetricUnavailableReplicas,
		value:      0,
	}, {
		key:        "replicaset:replicaset_test_1",
		metricType: core.MetricDesiredReplicas,
		value:      3,
	}, {
		key:        "replicaset:replicaset_test_1",
		metricType: core.MetricScheduledReplicas,
		value:      3,
	}, {
		key:        "replicaset:replicaset_test_1",
		metricType: core.MetricFullyLabeledReplicas,
		value:      3,
	}, {
		key:        "replicationcontroller:replicationcontroller_test_1",
		metricType: core.MetricDesiredReplicas,
		value:      5,
	}, {
		key:        "replicationcontroller:replicationcontroller_test_1",
		metricType: core.MetricScheduledReplicas,
		value:      5,
	}, {
		key:        "replicationcontroller:replicationcontroller_test_1",
		metricType: core.MetricFullyLabeledReplicas,
		value:      5,
	}, {
		key:        "daemonset:daemonset_test_1",
		metricType: core.MetricCurrentNumberScheduledDS,
		value:      6,
	}, {
		key:        "daemonset:daemonset_test_1",
		metricType: core.MetricNumberMisscheduledDS,
		value:      0,
	}, {
		key:        "daemonset:daemonset_test_1",
		metricType: core.MetricDesiredNumberScheduledDS,
		value:      6,
	}}

	// Test output
	for _, e := range expectations {
		m, ok := results.MetricSets[e.key]
		if !assert.True(t, ok, "missing metric %q", e.key) {
			continue
		}
		checkIntMetric(t, m, e.key, e.metricType, e.value)
	}

}
