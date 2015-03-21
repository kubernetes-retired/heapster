// Copyright 2014 Google Inc. All Rights Reserved.
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
	"strings"

	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/cache"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/fields"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/golang/glog"
)

// podsApi provides an interface to access all the pods that an instance of heapster
// needs to process.
// TODO(vishh): Add an interface to select specific nodes as part of the Watch.
type podsApi interface {
	// Returns a list of pods that exist on the nodes in 'nodeList'
	List(nodeList *nodes.NodeList) ([]api.Pod, error)

	// Returns debug information.
	DebugInfo() string
}

type realPodsApi struct {
	client *client.Client
	// a means to list all scheduled pods
	podLister *cache.StoreToPodLister
	reflector *cache.Reflector
	stopChan  chan struct{}
}

type podNodePair struct {
	pod      *kube_api.Pod
	nodeInfo *nodes.Info
}

func (self *realPodsApi) parsePod(podNodePair *podNodePair) *api.Pod {
	pod := podNodePair.pod
	node := podNodePair.nodeInfo
	localPod := api.Pod{
		Name:           pod.Name,
		Namespace:      pod.Namespace,
		ID:             string(pod.UID),
		Hostname:       pod.Status.Host,
		HostPublicIP:   pod.Status.HostIP,
		HostInternalIP: node.InternalIP,
		Status:         string(pod.Status.Phase),
		PodIP:          pod.Status.PodIP,
		Labels:         make(map[string]string, 0),
		Containers:     make([]api.Container, 0),
	}
	for key, value := range pod.Labels {
		localPod.Labels[key] = value
	}
	for _, container := range pod.Spec.Containers {
		localContainer := api.Container{}
		localContainer.Name = container.Name
		localPod.Containers = append(localPod.Containers, localContainer)
	}
	glog.V(5).Infof("parsed kube pod: %+v", localPod)

	return &localPod
}

func (self *realPodsApi) parseAllPods(podNodePairs []podNodePair) []api.Pod {
	out := make([]api.Pod, 0)
	for i := range podNodePairs {
		glog.V(5).Infof("Found kube Pod: %+v", podNodePairs[i].pod)
		out = append(out, *self.parsePod(&podNodePairs[i]))
	}

	return out
}

func (self *realPodsApi) getNodeSelector(nodeList *nodes.NodeList) (labels.Selector, error) {
	nodeLabels := []string{}
	for host := range nodeList.Items {
		nodeLabels = append(nodeLabels, fmt.Sprintf("DesiredState.Host==%s", host))
	}
	glog.V(2).Infof("using labels %v to find pods", nodeLabels)
	return labels.ParseSelector(strings.Join(nodeLabels, ","))
}

// Returns a map of minion hostnames to the Pods running in them.
func (self *realPodsApi) List(nodeList *nodes.NodeList) ([]api.Pod, error) {
	pods, err := self.podLister.List(labels.Everything())
	if err != nil {
		return []api.Pod{}, err
	}
	glog.V(5).Infof("got pods from api server %+v", pods)
	selectedPods := []podNodePair{}
	// TODO(vishh): Avoid this loop by setting a node selector on the watcher.
	for i, pod := range pods {
		if nodeInfo, ok := nodeList.Items[nodes.Host(pod.Status.Host)]; ok {
			selectedPods = append(selectedPods, podNodePair{&pods[i], &nodeInfo})
		} else {
			glog.V(2).Infof("pod %q with host %q and hostip %q not found in nodeList", pod.Name, pod.Status.Host, pod.Status.HostIP)
		}
	}
	glog.V(4).Infof("selected pods from api server %+v", selectedPods)

	return self.parseAllPods(selectedPods), nil
}

func (self *realPodsApi) DebugInfo() string {
	return ""
}

func newPodsApi(client *client.Client) podsApi {
	// Extend the selector to include specific nodes to monitor
	// or provide an API to update the nodes to monitor.
	selector, err := fields.ParseSelector("DesiredState.Host!=")
	if err != nil {
		panic(err)
	}

	lw := cache.NewListWatchFromClient(client, "pods", kube_api.NamespaceAll, selector)
	podLister := &cache.StoreToPodLister{Store: cache.NewStore(cache.MetaNamespaceKeyFunc)}
	// Watch and cache all running pods.
	reflector := cache.NewReflector(lw, &kube_api.Pod{}, podLister.Store, 0)
	stopChan := make(chan struct{})
	reflector.RunUntil(stopChan)

	podsApi := &realPodsApi{
		client:    client,
		podLister: podLister,
		stopChan:  stopChan,
		reflector: reflector,
	}

	return podsApi
}
