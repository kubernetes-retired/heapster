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

package app

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/heapster/metrics/apis/metrics"
	_ "k8s.io/heapster/metrics/apis/metrics/install"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sinks/metric"
	"k8s.io/heapster/metrics/storage/util"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
)

type MetricStorage struct {
	groupResource unversioned.GroupResource
	metricSink    *metricsink.MetricSink
	nodeLister    *cache.StoreToNodeLister
}

func NewStorage(groupResource unversioned.GroupResource, metricSink *metricsink.MetricSink, nodeLister *cache.StoreToNodeLister) *MetricStorage {
	return &MetricStorage{
		groupResource: groupResource,
		metricSink:    metricSink,
		nodeLister:    nodeLister,
	}
}

// Storage interface
func (m *MetricStorage) New() runtime.Object {
	return &metrics.NodeMetrics{}
}

// KindProvider interface
func (m *MetricStorage) Kind() string {
	return "NodeMetrics"
}

// Lister interface
func (m *MetricStorage) NewList() runtime.Object {
	return &metrics.NodeMetricsList{}
}

// Lister interface
func (m *MetricStorage) List(ctx api.Context, options *api.ListOptions) (runtime.Object, error) {
	labelSelector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		labelSelector = options.LabelSelector
	}
	nodes, err := m.nodeLister.NodeCondition(func(node *api.Node) bool {
		if labelSelector.Empty() {
			return true
		}
		return labelSelector.Matches(labels.Set(node.Labels))
	}).List()
	if err != nil {
		errMsg := fmt.Errorf("Error while listing nodes: %v", err)
		glog.Error(errMsg)
		return &metrics.NodeMetricsList{}, errMsg
	}

	res := metrics.NodeMetricsList{}
	for _, node := range nodes {
		if m := m.getNodeMetrics(node.Name); m != nil {
			res.Items = append(res.Items, *m)
		}
	}
	return &res, nil
}

// Getter interface
func (m *MetricStorage) Get(ctx api.Context, name string) (runtime.Object, error) {
	nodeMetrics := m.getNodeMetrics(name)
	if nodeMetrics == nil {
		return &metrics.NodeMetrics{}, errors.NewNotFound(m.groupResource, name)
	}
	return nodeMetrics, nil
}

func (m *MetricStorage) getNodeMetrics(node string) *metrics.NodeMetrics {
	batch := m.metricSink.GetLatestDataBatch()
	if batch == nil {
		return nil
	}

	ms, found := batch.MetricSets[core.NodeKey(node)]
	if !found {
		return nil
	}

	usage, err := util.ParseResourceList(ms)
	if err != nil {
		return nil
	}

	return &metrics.NodeMetrics{
		ObjectMeta: api.ObjectMeta{
			Name:              node,
			CreationTimestamp: unversioned.NewTime(time.Now()),
		},
		Timestamp: unversioned.NewTime(batch.Timestamp),
		Window:    unversioned.Duration{Duration: time.Minute},
		Usage:     usage,
	}
}
