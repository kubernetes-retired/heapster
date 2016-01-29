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
	"net/url"
	"time"

	"github.com/golang/glog"

	kube_config "k8s.io/heapster/common/kubernetes"
	"k8s.io/heapster/metrics/util"

	"k8s.io/heapster/metrics/core"
	kube_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	kube_client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

type PodBasedEnricher struct {
	podLister *cache.StoreToPodLister
	reflector *cache.Reflector
}

func (this *PodBasedEnricher) Name() string {
	return "pod_based_enricher"
}

func (this *PodBasedEnricher) Process(batch *core.DataBatch) (*core.DataBatch, error) {
	pods, err := this.podLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		if pod.Status.Phase == kube_api.PodRunning {
			addPodInfo(pod, batch)
		}
	}
	return batch, nil
}

func addPodInfo(pod *kube_api.Pod, batch *core.DataBatch) {
	podMs, found := batch.MetricSets[core.PodKey(pod.Namespace, pod.Name)]
	if !found {
		// create pod stub.
		glog.V(2).Infof("Pod not found: %s", core.PodKey(pod.Namespace, pod.Name))
		podMs = &core.MetricSet{
			MetricValues: make(map[string]core.MetricValue),
			Labels: map[string]string{
				core.LabelMetricSetType.Key: core.MetricSetTypePod,
				core.LabelNamespaceName.Key: pod.Namespace,
				core.LabelPodNamespace.Key:  pod.Namespace,
				core.LabelPodName.Key:       pod.Name,
			},
		}
		batch.MetricSets[core.PodKey(pod.Namespace, pod.Name)] = podMs
	}
	// Add UID to pod
	podMs.Labels[core.LabelPodId.Key] = string(pod.UID)
	podMs.Labels[core.LabelLabels.Key] = util.LabelsToString(pod.Labels, ",")

	// Add cpu/mem requests and limits to containers
	for _, container := range pod.Spec.Containers {
		requests := container.Resources.Requests
		limits := container.Resources.Limits

		cpuRequestMilli := int64(0)
		memRequest := int64(0)
		cpuLimitMilli := int64(0)
		memLimit := int64(0)

		if val, found := requests[kube_api.ResourceCPU]; found {
			cpuRequestMilli = val.MilliValue()
		}
		if val, found := requests[kube_api.ResourceMemory]; found {
			memRequest = val.Value()
		}

		if val, found := limits[kube_api.ResourceCPU]; found {
			cpuLimitMilli = val.MilliValue()
		}
		if val, found := limits[kube_api.ResourceMemory]; found {
			memLimit = val.Value()
		}

		containerKey := core.PodContainerKey(pod.Namespace, pod.Name, container.Name)
		containerMs, found := batch.MetricSets[containerKey]
		if !found {
			glog.V(2).Infof("Container %s not found, creating a stub", containerKey)
			containerMs = &core.MetricSet{
				MetricValues: make(map[string]core.MetricValue),
				Labels: map[string]string{
					core.LabelMetricSetType.Key:      core.MetricSetTypePodContainer,
					core.LabelNamespaceName.Key:      pod.Namespace,
					core.LabelPodNamespace.Key:       pod.Namespace,
					core.LabelPodName.Key:            pod.Name,
					core.LabelContainerName.Key:      container.Name,
					core.LabelContainerBaseImage.Key: container.Image,
				},
			}
			batch.MetricSets[containerKey] = containerMs
		}

		containerMs.MetricValues[core.MetricCpuRequest.Name] = intValue(cpuRequestMilli)
		containerMs.MetricValues[core.MetricMemoryRequest.Name] = intValue(memRequest)
		containerMs.MetricValues[core.MetricCpuLimit.Name] = intValue(cpuLimitMilli)
		containerMs.MetricValues[core.MetricMemoryLimit.Name] = intValue(memLimit)

		containerMs.Labels[core.LabelPodId.Key] = string(pod.UID)
		containerMs.Labels[core.LabelLabels.Key] = util.LabelsToString(pod.Labels, ",")
	}
}

func intValue(value int64) core.MetricValue {
	return core.MetricValue{
		IntValue:   value,
		MetricType: core.MetricGauge,
		ValueType:  core.ValueInt64,
	}
}

func NewPodBasedEnricher(url *url.URL) (*PodBasedEnricher, error) {
	kubeConfig, err := kube_config.GetKubeClientConfig(url)
	if err != nil {
		return nil, err
	}
	kubeClient := kube_client.NewOrDie(kubeConfig)

	// watch nodes
	lw := cache.NewListWatchFromClient(kubeClient, "pods", kube_api.NamespaceAll, fields.Everything())
	podLister := &cache.StoreToPodLister{Store: cache.NewStore(cache.MetaNamespaceKeyFunc)}
	reflector := cache.NewReflector(lw, &kube_api.Pod{}, podLister.Store, time.Hour)
	reflector.Run()

	return &PodBasedEnricher{
		podLister: podLister,
		reflector: reflector,
	}, nil
}
