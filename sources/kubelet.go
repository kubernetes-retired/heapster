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

package sources

import (
	"fmt"
	"net/url"
	"time"

	. "k8s.io/heapster/core"
	"k8s.io/heapster/sources/api"

	"github.com/golang/glog"
	kube_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	kube_client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
)

// TODO: following constants are copied from k8s, change to use them directly
const (
	kubernetesPodNameLabel      = "io.kubernetes.pod.name"
	kubernetesPodNamespaceLabel = "io.kubernetes.pod.namespace"
	kubernetesPodUID            = "io.kubernetes.pod.uid"
)

// Kubelet-provided metrics for pod and system container.
type kubeletMetricsSource struct {
	host          Host
	kubeletClient *KubeletClient
	hostname      string
	hostId        string
}

func (this *kubeletMetricsSource) String() string {
	return fmt.Sprintf("kubelet:%s:%d", this.host.IP, this.host.Port)
}

func (this *kubeletMetricsSource) decodeMetrics(c *api.Container) (string, *MetricSet) {
	var metricSetKey string
	cMetrics := &MetricSet{
		MetricValues: map[string]MetricValue{},
		Labels:       map[string]string{},
	}

	for _, metric := range StandardMetrics {
		if metric.HasValue(&c.Spec) {
			m := metric.GetValue(&c.Spec, c.Stats[0])
			m.MetricType = metric.Type
			m.ValueType = metric.ValueType
			cMetrics.MetricValues[metric.Name] = m
		}
	}

	if isNode(c) {
		metricSetKey = NodeKey(this.hostname)
		cMetrics.Labels[LabelMetricSetType.Key] = MetricSetTypeNode
	} else if isSysContainer(c) {
		cName := getSysContainerName(c)
		metricSetKey = NodeContainerKey(this.hostname, cName)
		cMetrics.Labels[LabelMetricSetType.Key] = MetricSetTypeSystemContainer
		cMetrics.Labels[LabelContainerName.Key] = cName
	} else {
		// TODO: figure out why "io.kubernetes.container.name" is not set
		cName := c.Name
		ns := c.Spec.Labels[kubernetesPodNamespaceLabel]
		podName := c.Spec.Labels[kubernetesPodNameLabel]
		metricSetKey = PodContainerKey(ns, podName, cName)
		cMetrics.Labels[LabelMetricSetType.Key] = MetricSetTypePodContainer
		cMetrics.Labels[LabelContainerName.Key] = cName
		cMetrics.Labels[LabelPodId.Key] = c.Spec.Labels[kubernetesPodUID]
		cMetrics.Labels[LabelPodName.Key] = podName
		cMetrics.Labels[LabelPodNamespace.Key] = ns
	}

	// common labels
	cMetrics.Labels[LabelHostname.Key] = this.hostname
	cMetrics.Labels[LabelHostID.Key] = this.hostId

	// TODO: add labels: LabelPodNamespaceUID, LabelLabels, LabelResourceID, LabelContainerBaseImage

	return metricSetKey, cMetrics
}

func (this *kubeletMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	containers, err := this.kubeletClient.GetAllRawContainers(this.host, start, end)
	if err != nil {
		glog.Errorf("error while getting containers from Kubelet: %v", err)
	}
	glog.Infof("successfully obtained stats for %v containers", len(containers))

	result := &DataBatch{
		Timestamp:  end,
		MetricSets: map[string]*MetricSet{},
	}
	for _, c := range containers {
		name, metrics := this.decodeMetrics(&c)
		result.MetricSets[name] = metrics
	}
	return result
}

type kubeletProvider struct {
	nodeLister    *cache.StoreToNodeLister
	reflector     *cache.Reflector
	kubeletClient *KubeletClient
}

func (this *kubeletProvider) GetMetricsSources() []MetricsSource {
	sources := []MetricsSource{}
	nodes, err := this.nodeLister.List()
	if err != nil {
		glog.Errorf("error while listing nodes: %v", err)
		return sources
	}
	for _, node := range nodes.Items {
		hostname, ip, err := getNodeHostnameAndIP(&node)
		if err != nil {
			glog.Errorf("%v", err)
			continue
		}
		sources = append(sources, &kubeletMetricsSource{
			host:          Host{IP: ip, Port: this.kubeletClient.GetPort()},
			kubeletClient: this.kubeletClient,
			hostname:      hostname,
			hostId:        node.Spec.ExternalID,
		})
	}
	return sources
}

func getNodeHostnameAndIP(node *kube_api.Node) (string, string, error) {
	for _, c := range node.Status.Conditions {
		if c.Type == kube_api.NodeReady && c.Status != kube_api.ConditionTrue {
			return "", "", fmt.Errorf("Node %v is not ready", node.Name)
		}
	}
	hostname, ip := node.Name, ""
	for _, addr := range node.Status.Addresses {
		if addr.Type == kube_api.NodeHostName && addr.Address != "" {
			hostname = addr.Address
		}
		if addr.Type == kube_api.NodeInternalIP && addr.Address != "" {
			ip = addr.Address
		}
	}
	if ip != "" {
		return hostname, ip, nil
	}
	return "", "", fmt.Errorf("Node %v has no valid hostname and/or IP address: %v %v", node.Name, hostname, ip)
}

func NewKubeletProvider(uri *url.URL) (MetricsSourceProvider, error) {
	// create clients
	kubeConfig, kubeletConfig, err := getKubeConfigs(uri)
	if err != nil {
		return nil, err
	}
	kubeClient := kube_client.NewOrDie(kubeConfig)
	kubeletClient, err := NewKubeletClient(kubeletConfig)
	if err != nil {
		return nil, err
	}
	// watch nodes
	lw := cache.NewListWatchFromClient(kubeClient, "nodes", kube_api.NamespaceAll, fields.Everything())
	nodeLister := &cache.StoreToNodeLister{Store: cache.NewStore(cache.MetaNamespaceKeyFunc)}
	reflector := cache.NewReflector(lw, &kube_api.Node{}, nodeLister.Store, 0)
	reflector.Run()

	return &kubeletProvider{
		nodeLister:    nodeLister,
		reflector:     reflector,
		kubeletClient: kubeletClient,
	}, nil
}
