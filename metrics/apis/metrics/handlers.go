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

// The file path is compatible with Kubernetes standards. This not a requirement
// right now but in the future we want to reuse apiserver code, which
// requires it.

package metrics

import (
	"fmt"
	"time"

	restful "github.com/emicklei/go-restful"

	"k8s.io/heapster/metrics/apis/metrics/v1alpha1"
	"k8s.io/heapster/metrics/core"
	metricsink "k8s.io/heapster/metrics/sinks/metric"
	"k8s.io/kubernetes/pkg/api/resource"
	kube_unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	kube_v1 "k8s.io/kubernetes/pkg/api/v1"
)

type Api struct {
	metricSink *metricsink.MetricSink
}

func NewApi(metricSink *metricsink.MetricSink) *Api {
	return &Api{metricSink: metricSink}
}

// TODO(piosz): add support for label selector
// TODO(piosz): add support for all-namespaces
// TODO(piosz): add error handling
func (a *Api) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/apis/metrics/v1alpha1").
		Doc("Root endpoint of metrics API").
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/nodes/").
		To(a.nodeMetricsList).
		Doc("Get a list of metrics for all available nodes.").
		Operation("nodeMetricsList"))

	ws.Route(ws.GET("/nodes/{node-name}/").
		To(a.nodeMetrics).
		Doc("Get a list of all available metrics for the specified node.").
		Operation("nodeMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")))

	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/").
		To(a.podMetricsList).
		Doc("Get a list of metrics for all available pods in the specified namespace.").
		Operation("podMetricsList").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/").
		To(a.podMetrics).
		Doc("Get metrics for the specified pod in the specified namespace.").
		Operation("podMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

	container.Add(ws)
}

func (a *Api) nodeMetricsList(request *restful.Request, response *restful.Response) {
	res := []*v1alpha1.NodeMetrics{}
	for _, node := range a.metricSink.GetNodes() {
		if m := a.getNodeMetrics(node); m != nil {
			res = append(res, m)
		}
	}
	response.WriteEntity(res)
}

func (a *Api) nodeMetrics(request *restful.Request, response *restful.Response) {
	node := request.PathParameter("node-name")
	response.WriteEntity(a.getNodeMetrics(node))
}

func (a *Api) getNodeMetrics(node string) *v1alpha1.NodeMetrics {
	batch := a.metricSink.GetLatestDataBatch()
	if batch == nil {
		return nil
	}

	ms, found := batch.MetricSets[core.NodeKey(node)]
	if !found {
		return nil
	}

	usage, err := parseResourceList(ms)
	if err != nil {
		return nil
	}

	return &v1alpha1.NodeMetrics{
		ObjectMeta: kube_v1.ObjectMeta{
			Name:              node,
			CreationTimestamp: kube_unversioned.NewTime(time.Now()),
		},
		Timestamp: kube_unversioned.NewTime(batch.Timestamp),
		Window:    kube_unversioned.Duration{Duration: time.Minute},
		Usage:     usage,
	}
}

func parseResourceList(ms *core.MetricSet) (kube_v1.ResourceList, error) {
	cpu, found := ms.MetricValues[core.MetricCpuUsageRate.MetricDescriptor.Name]
	if !found {
		return kube_v1.ResourceList{}, fmt.Errorf("cpu not found")
	}
	mem, found := ms.MetricValues[core.MetricMemoryWorkingSet.MetricDescriptor.Name]
	if !found {
		return kube_v1.ResourceList{}, fmt.Errorf("memory not found")
	}

	return kube_v1.ResourceList{
		kube_v1.ResourceCPU: *resource.NewMilliQuantity(
			cpu.IntValue,
			resource.DecimalSI),
		kube_v1.ResourceMemory: *resource.NewQuantity(
			mem.IntValue,
			resource.BinarySI),
	}, nil
}

func (a *Api) podMetricsList(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace-name")
	res := []*v1alpha1.PodMetrics{}
	for _, pod := range a.metricSink.GetPodsFromNamespace(ns) {
		if m := a.getPodMetrics(ns, pod); m != nil {
			res = append(res, m)
		}
	}
	response.WriteEntity(res)
}

func (a *Api) podMetrics(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace-name")
	pod := request.PathParameter("pod-name")
	response.WriteEntity(a.getPodMetrics(ns, pod))
}

// TODO(piosz): the implementation is inefficient. Fix this once adding support for labelSelector.
func (a *Api) getPodMetrics(ns, pod string) *v1alpha1.PodMetrics {
	batch := a.metricSink.GetLatestDataBatch()
	if batch == nil {
		return nil
	}

	res := &v1alpha1.PodMetrics{
		ObjectMeta: kube_v1.ObjectMeta{
			Name:              pod,
			Namespace:         ns,
			CreationTimestamp: kube_unversioned.NewTime(time.Now()),
		},
		Timestamp:  kube_unversioned.NewTime(batch.Timestamp),
		Window:     kube_unversioned.Duration{Duration: time.Minute},
		Containers: make([]v1alpha1.ContainerMetrics, 0),
	}

	for _, c := range a.metricSink.GetContainersForPodFromNamespace(ns, pod) {
		ms, found := batch.MetricSets[core.PodContainerKey(ns, pod, c)]
		if !found {
			continue
		}

		usage, err := parseResourceList(ms)
		if err != nil {
			return nil
		}

		res.Containers = append(res.Containers, v1alpha1.ContainerMetrics{Name: c, Usage: usage})
	}

	return res
}
