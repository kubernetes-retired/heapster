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
	"strconv"

	"github.com/GoogleCloudPlatform/heapster/extpoints"
	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/sources/datasource"
	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/golang/glog"
)

const (
	defaultKubeletPort  = 10255
	defaultKubeletHttps = false
)

func init() {
	extpoints.SourceFactories.Register(CreateKubeSources, "kubernetes")
}

func CreateKubeSources(uri string, options map[string][]string) ([]api.Source, error) {
	kubeConfig, err := kube_client.InClusterConfig()

	if err != nil {
		return nil, err
	}

	kubeClient := kube_client.NewOrDie(kubeConfig)

	nodesApi, err := nodes.NewKubeNodes(kubeClient)
	if err != nil {
		return nil, err
	}
	kubeletPort := defaultKubeletPort
	if len(options["kubeletPort"]) >= 1 {
		kubeletPort, err = strconv.Atoi(options["kubeletPort"][0])
		if err != nil {
			return nil, err
		}
	}

	kubeletHttps := defaultKubeletHttps
	if len(options["kubeletHttps"]) >= 1 {
		kubeletHttps, err = strconv.ParseBool(options["kubeletHttps"][0])
		if err != nil {
			return nil, err
		}
	}
	glog.Infof("Using Kubernetes client with master %q and version %q\n", kubeConfig.Host, kubeConfig.Version)
	glog.Infof("Using kubelet port %d", kubeletPort)

	kubeletConfig := &kube_client.KubeletConfig{
		Port:            uint(kubeletPort),
		EnableHttps:     kubeletHttps,
		TLSClientConfig: kubeConfig.TLSClientConfig,
	}

	kubeletApi, err := datasource.NewKubelet(kubeletConfig)
	if err != nil {
		return nil, err
	}

	kubePodsSource := NewKubePodMetrics(kubeletPort, kubeletApi, nodesApi, newPodsApi(kubeClient))
	kubeNodeSource := NewKubeNodeMetrics(kubeletPort, kubeletApi, nodesApi)
	kubeEventsSource := NewKubeEvents(kubeClient)

	return []api.Source{kubePodsSource, kubeNodeSource, kubeEventsSource}, nil
}
