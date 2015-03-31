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
	"github.com/GoogleCloudPlatform/heapster/sources/datasource"
	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/clientauth"
	"github.com/golang/glog"
)

func CreateKubeSources(master, kubeVersion, clientAuthFile, kubeletPort string, masterInsecure bool) ([]api.Source, error) {
	if len(master) == 0 {
		return nil, fmt.Errorf("kubernetes_master flag not specified")
	}
	if kubeVersion == "" {
		return nil, fmt.Errorf("kubernetes API version invalid")
	}
	if !(strings.HasPrefix(master, "http://") || strings.HasPrefix(master, "https://")) {
		master = "http://" + master
	}

	kubeConfig := kube_client.Config{
		Host:     master,
		Version:  kubeVersion,
		Insecure: masterInsecure,
	}

	if len(clientAuthFile) > 0 {
		clientAuth, err := clientauth.LoadFromFile(clientAuthFile)
		if err != nil {
			return nil, err
		}

		kubeConfig, err = clientAuth.MergeWithConfig(kubeConfig)
		if err != nil {
			return nil, err
		}
	}

	kubeClient := kube_client.NewOrDie(&kubeConfig)

	nodesApi, err := nodes.NewKubeNodes(kubeClient)
	if err != nil {
		return nil, err
	}
	glog.Infof("Using Kubernetes client with master %q and version %s\n", master, kubeVersion)
	glog.Infof("Using kubelet port %q", kubeletPort)
	kubeletApi := datasource.NewKubelet()
	kubePodsSource := NewKubePodMetrics(kubeletPort, nodesApi, newPodsApi(kubeClient), kubeletApi)
	kubeNodeSource := NewKubeNodeMetrics(kubeletPort, kubeletApi, nodesApi)
	kubeEventSource := NewKubeEvents(kubeClient)

	return []api.Source{kubePodsSource, kubeNodeSource, kubeEventSource}, nil
}
