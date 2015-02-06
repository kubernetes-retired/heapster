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
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info"
)

const (
	// Cadvisor port in kubernetes.
	cadvisorPort = 4194

	kubeClientVersion = "v1beta1"
)

var (
	argMaster      = flag.String("kubernetes_master", "", "Kubernetes master IP")
	argMasterInsecure = flag.Bool("kubernetes_insecure", true, "Trust Kubernetes master certificate (if using https)")
	argKubeletPort = flag.String("kubelet_port", "10250", "Kubelet port")
)

type kubeSource struct {
	client       *kube_client.Client
	kubeletPort  string
	pollDuration time.Duration
	nodesApi    nodes.NodesApi
	podsApi     podsApi
	stateLock    sync.RWMutex
	podErrors    map[podInstance]int // guarded by stateLock
	lastQuery time.Time
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

	podInstance := podInstance{name: pod.Name, id: pod.ID, ip: pod.HostIP}
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

func (self *kubeSource) getNumStatsToFetch() int {
	numStats := int(self.pollDuration / time.Second)
	if time.Since(self.lastQuery) > self.pollDuration {
		numStats = int(time.Since(self.lastQuery)/time.Second)
	}
	return numStats
}

func (self *kubeSource) getStatsFromKubelet(pod Pod, containerName string) (cadvisor.ContainerSpec, []*cadvisor.ContainerStats, error) {
	var containerInfo cadvisor.ContainerInfo
	values := url.Values{}
	values.Add("num_stats", strconv.Itoa(self.getNumStatsToFetch()))
	url := "http://" + pod.HostIP + ":" + self.kubeletPort + filepath.Join("/stats", pod.Namespace, pod.Name, pod.ID, containerName) + "?" + values.Encode()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return cadvisor.ContainerSpec{}, []*cadvisor.ContainerStats{}, err
	}
	err = PostRequestAndGetValue(&http.Client{}, req, &containerInfo)
	if err != nil {
		glog.Errorf("failed to get stats from kubelet url: %s - %s\n", url, err)
		self.recordPodError(pod)
		return cadvisor.ContainerSpec{}, []*cadvisor.ContainerStats{}, nil
	}

	return containerInfo.Spec, containerInfo.Stats, nil
}

func (self *kubeSource) getNodesInfo(nodeList *nodes.NodeList) ([]RawContainer, error) {
	glog.V(2).Infof("current nodes %+v", nodeList)
	nodesInfo := []RawContainer{}
	for node := range nodeList.Items {
		spec, stats, err := self.getStatsFromKubelet(Pod{HostIP: node.Name}, "/")
		if err != nil {
			glog.V(1).Infof("Failed to get machine stats from kubelet for node %s", node)
			return []RawContainer{}, err
		}
		if len(stats) > 0 {
			container := RawContainer{
				Hostname:  node.Name,
				Container: Container{"/", spec, stats},
			}
			nodesInfo = append(nodesInfo, container)
		}
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
			spec, stats, err := self.getStatsFromKubelet(pod, container.Name)
			if err != nil {
				// Containers could be in the process of being setup or restarting while the pod is alive.
				glog.Errorf("failed to get stats for container %q/%q in pod %q", container.Name, pod.Namespace, pod.Name)
				continue
			}
			glog.V(2).Infof("Fetched stats from kubelet for container %s in pod %s", container.Name, pod.Name)
			container.Stats = stats
			container.Spec = spec
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
		client:      kubeClient,
		lastQuery:   time.Now(),
		pollDuration: pollDuration,
		kubeletPort: *argKubeletPort,
		nodesApi:    nodesApi,
		podsApi:     newPodsApi(kubeClient),
		podErrors:   make(map[podInstance]int),
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
