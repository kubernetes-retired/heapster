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
	"k8s.io/heapster/sources/datasource"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	kube_client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
)

// Kubelet-provided metrics for pod and system container.
type kubeletMetricsSource struct {
	host          datasource.Host
	kubeletClient *datasource.KubeletClient
}

func (this *kubeletMetricsSource) String() string {
	return fmt.Sprintf("kubelet:%s:%d", this.host.IP, this.host.Port)
}

func (this *kubeletMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	cont, err := this.kubeletClient.GetAllRawContainers(this.host, start, end)
	if err != nil {
		glog.Errorf("error while getting containers from Kubelet: %v", err)
	}
	glog.Infof("successfully obtained stats for %v containers", len(cont))
	var tmp DataBatch
	return &tmp
}

type kubeletProvider struct {
	nodeLister    *cache.StoreToNodeLister
	reflector     *cache.Reflector
	kubeletClient *datasource.KubeletClient
}

func (this *kubeletProvider) GetMetricsSources() []MetricsSource {
	sources := []MetricsSource{}
	nodes, err := this.nodeLister.List()
	if err != nil {
		glog.Errorf("error while listing nodes: %v", err)
		return sources
	}
	for _, node := range nodes.Items {
		addr, err := getNodeAddr(&node)
		if err != nil {
			glog.Errorf("%v", err)
		}
		sources = append(sources, &kubeletMetricsSource{
			host:          datasource.Host{IP: addr, Port: this.kubeletClient.GetPort()},
			kubeletClient: this.kubeletClient,
		})
	}
	return sources
}

func getNodeAddr(node *api.Node) (string, error) {
	for _, c := range node.Status.Conditions {
		if c.Type == api.NodeReady && c.Status != api.ConditionTrue {
			return "", fmt.Errorf("Node %v is not ready", node.Name)
		}
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == api.NodeInternalIP && addr.Address != "" {
			return addr.Address, nil
		}
	}
	return "", fmt.Errorf("Node %v has no valid IP address", node.Name)
}

func NewKubeletProvider(uri *url.URL) (MetricsSourceProvider, error) {
	// create clients
	kubeConfig, kubeletConfig, err := getKubeConfigs(uri)
	if err != nil {
		return nil, err
	}
	kubeClient := kube_client.NewOrDie(kubeConfig)
	kubeletClient, err := datasource.NewKubeletClient(kubeletConfig)
	if err != nil {
		return nil, err
	}
	// watch nodes
	lw := cache.NewListWatchFromClient(kubeClient, "nodes", api.NamespaceAll, fields.Everything())
	nodeLister := &cache.StoreToNodeLister{Store: cache.NewStore(cache.MetaNamespaceKeyFunc)}
	reflector := cache.NewReflector(lw, &api.Node{}, nodeLister.Store, 0)
	reflector.Run()

	return &kubeletProvider{
		nodeLister:    nodeLister,
		reflector:     reflector,
		kubeletClient: kubeletClient,
	}, nil
}
