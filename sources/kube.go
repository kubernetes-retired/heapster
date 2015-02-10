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
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/datasource"
	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/golang/glog"
)

const (
	// Cadvisor port in kubernetes.
	cadvisorPort = 4194

	kubeClientVersion = "v1beta1"
)

var (
	argMaster         = flag.String("kubernetes_master", "", "Kubernetes master IP")
	argMasterInsecure = flag.Bool("kubernetes_insecure", true, "Trust Kubernetes master certificate (if using https)")
	argKubeletPort    = flag.String("kubelet_port", "10250", "Kubelet port")
)

type kubeSource struct {
	kubeletPort  string
	pollDuration time.Duration
	nodesApi     nodes.NodesApi
	podsApi      podsApi
	kubeletApi   datasource.Kubelet
	stateLock    sync.RWMutex
	podErrors    map[podInstance]int // guarded by stateLock
	lastQuery    time.Time
}

type podInstance struct {
	name string
	id   string
	ip   string
}

func (self *kubeSource) recordPodError(pod Pod) {
	// Heapster knows about pods before they are up and running on a node.
	// Ignore errors for Pods that are not Running.
	if pod.Status != "Running" {
		return
	}

	self.stateLock.Lock()
	defer self.stateLock.Unlock()

	podInstance := podInstance{name: pod.Name, id: pod.ID, ip: pod.HostPublicIP}
	self.podErrors[podInstance]++
}

func (self *kubeSource) getState() string {
	self.stateLock.RLock()
	defer self.stateLock.RUnlock()

	state := "\tHealthy Nodes:\n"
	if len(self.podErrors) != 0 {
		state += fmt.Sprintf("\tPod Errors: %+v\n", self.podErrors)
	} else {
		state += "\tNo pod errors\n"
	}
	return state
}

func (self *kubeSource) numStatsToFetch() int {
	numStats := int(self.pollDuration / time.Second)
	if time.Since(self.lastQuery) > self.pollDuration {
		numStats = int(time.Since(self.lastQuery) / time.Second)
	}
	return numStats
}

func (self *kubeSource) getStatsFromKubelet(pod Pod, containerName string) (*datasource.Container, error) {
	resource := filepath.Join("/stats", pod.Name, containerName)
	if containerName == "/" {
		resource += "/"
	}

	return self.kubeletApi.GetContainer(datasource.Host{IP: pod.HostInternalIP, Port: self.kubeletPort, Resource: resource}, self.numStatsToFetch())
}

func (self *kubeSource) getNodesInfo(nodeList *nodes.NodeList) ([]Container, error) {
	nodesInfo := []Container{}
	for host, info := range nodeList.Items {
		rawContainer, err := self.getStatsFromKubelet(Pod{HostInternalIP: info.InternalIP}, "/")
		if err != nil {
			glog.V(1).Infof("Failed to get machine stats from kubelet for node %s", host)
			return nil, err
		}
		if rawContainer == nil {
			// no stats found.
			glog.V(1).Infof("no machine stats from kubelet on node %s", host)
			continue
		}
		container := Container{
			Hostname: string(host),
			Name:     rawContainer.Name,
			Spec:     rawContainer.Spec,
			Stats:    rawContainer.Stats,
		}
		nodesInfo = append(nodesInfo, container)
	}

	return nodesInfo, nil
}

func (self *kubeSource) getPodInfo(nodeList *nodes.NodeList) ([]Pod, error) {
	pods, err := self.podsApi.List(nodeList)
	if err != nil {
		return []Pod{}, err
	}
	for _, pod := range pods {
		for _, container := range pod.Containers {
			rawContainer, err := self.getStatsFromKubelet(pod, container.Name)
			if err != nil {
				// Containers could be in the process of being setup or restarting while the pod is alive.
				glog.Errorf("failed to get stats for container %q/%q in pod %q", container.Name, pod.Namespace, pod.Name)
				continue
			}
			glog.V(2).Infof("Fetched stats from kubelet for container %s in pod %s", container.Name, pod.Name)
			container.Stats = rawContainer.Stats
			container.Spec = rawContainer.Spec
		}
	}

	return pods, nil
}

func (self *kubeSource) GetInfo() (ContainerData, error) {
	kubeNodes, err := self.nodesApi.List()
	if err != nil || len(kubeNodes.Items) == 0 {
		return ContainerData{}, err
	}
	podsInfo, err := self.getPodInfo(kubeNodes)
	if err != nil {
		return ContainerData{}, err
	}

	nodesInfo, err := self.getNodesInfo(kubeNodes)
	if err != nil {
		return ContainerData{}, err
	}
	glog.V(2).Info("Fetched list of nodes from the master")
	self.lastQuery = time.Now()

	return ContainerData{Pods: podsInfo, Machine: nodesInfo}, nil
}

func newKubeSource(pollDuration time.Duration) (*kubeSource, error) {
	if len(*argMaster) == 0 {
		return nil, fmt.Errorf("kubernetes_master flag not specified")
	}

	if !(strings.HasPrefix(*argMaster, "http://") || strings.HasPrefix(*argMaster, "https://")) {
		*argMaster = "http://" + *argMaster
	}

	kubeClient := kube_client.NewOrDie(&kube_client.Config{
		Host:     *argMaster,
		Version:  kubeClientVersion,
		Insecure: *argMasterInsecure,
	})

	nodesApi, err := nodes.NewKubeNodes(kubeClient)
	if err != nil {
		return nil, err
	}
	glog.Infof("Using Kubernetes client with master %q and version %s\n", *argMaster, kubeClientVersion)
	glog.Infof("Using kubelet port %q", *argKubeletPort)

	return &kubeSource{
		lastQuery:    time.Now(),
		pollDuration: pollDuration,
		kubeletPort:  *argKubeletPort,
		nodesApi:     nodesApi,
		podsApi:      newPodsApi(kubeClient),
		podErrors:    make(map[podInstance]int),
	}, nil
}

func (self *kubeSource) DebugInfo() string {
	desc := "Source type: Kube\n"
	desc += fmt.Sprintf("\tClient config: master ip %q, version %s\n", *argMaster, kubeClientVersion)
	desc += fmt.Sprintf("\tUsing kubelet port %q\n", self.kubeletPort)
	desc += self.getState()
	desc += "\n"
	desc += self.nodesApi.DebugInfo() + "\n"
	return desc
}
