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
	podLister     *cache.StoreToPodLister
}

func NewStorage(groupResource unversioned.GroupResource, metricSink *metricsink.MetricSink, podLister *cache.StoreToPodLister) *MetricStorage {
	return &MetricStorage{
		groupResource: groupResource,
		metricSink:    metricSink,
		podLister:     podLister,
	}
}

// Storage interface
func (m *MetricStorage) New() runtime.Object {
	return &metrics.PodMetrics{}
}

// KindProvider interface
func (m *MetricStorage) Kind() string {
	return "PodMetrics"
}

// Lister interface
func (m *MetricStorage) NewList() runtime.Object {
	return &metrics.PodMetricsList{}
}

// Lister interface
func (m *MetricStorage) List(ctx api.Context, options *api.ListOptions) (runtime.Object, error) {
	labelSelector := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		labelSelector = options.LabelSelector
	}
	namespace := api.NamespaceValue(ctx)
	pods, err := m.podLister.Pods(namespace).List(labelSelector)
	if err != nil {
		errMsg := fmt.Errorf("Error while listing pods for selector %v: %v", labelSelector, err)
		glog.Error(errMsg)
		return &metrics.PodMetricsList{}, errMsg
	}

	res := metrics.PodMetricsList{}
	for _, pod := range pods {
		if podMetrics := m.getPodMetrics(pod); podMetrics != nil {
			res.Items = append(res.Items, *podMetrics)
		} else {
			glog.Infof("No metrics for pod %s/%s", pod.Namespace, pod.Name)
		}
	}
	return &res, nil
}

// Getter interface
func (m *MetricStorage) Get(ctx api.Context, name string) (runtime.Object, error) {
	namespace := api.NamespaceValue(ctx)

	pod, err := m.podLister.Pods(namespace).Get(name)
	if err != nil {
		errMsg := fmt.Errorf("Error while getting pod %v: %v", name, err)
		glog.Error(errMsg)
		return &metrics.PodMetrics{}, errMsg
	}
	if pod == nil {
		return &metrics.PodMetrics{}, errors.NewNotFound(api.Resource("Pod"), fmt.Sprintf("%v/%v", namespace, name))
	}

	podMetrics := m.getPodMetrics(pod)
	if podMetrics == nil {
		return &metrics.PodMetrics{}, errors.NewNotFound(m.groupResource, fmt.Sprintf("%v/%v", namespace, name))
	}
	return podMetrics, nil
}

func (m *MetricStorage) getPodMetrics(pod *api.Pod) *metrics.PodMetrics {
	batch := m.metricSink.GetLatestDataBatch()
	if batch == nil {
		return nil
	}

	res := &metrics.PodMetrics{
		ObjectMeta: api.ObjectMeta{
			Name:              pod.Name,
			Namespace:         pod.Namespace,
			CreationTimestamp: unversioned.NewTime(time.Now()),
		},
		Timestamp:  unversioned.NewTime(batch.Timestamp),
		Window:     unversioned.Duration{Duration: time.Minute},
		Containers: make([]metrics.ContainerMetrics, 0),
	}

	for _, c := range pod.Spec.Containers {
		ms, found := batch.MetricSets[core.PodContainerKey(pod.Namespace, pod.Name, c.Name)]
		if !found {
			glog.Infof("No metrics for container %s in pod %s/%s", c.Name, pod.Namespace, pod.Name)
			return nil
		}
		usage, err := util.ParseResourceList(ms)
		if err != nil {
			return nil
		}
		res.Containers = append(res.Containers, metrics.ContainerMetrics{Name: c.Name, Usage: usage})
	}

	return res
}
