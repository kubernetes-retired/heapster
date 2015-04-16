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
	"net/url"
	"os"
	"strconv"

	"github.com/GoogleCloudPlatform/heapster/extpoints"
	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/sources/datasource"
	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/clientauth"
	"github.com/golang/glog"
)

const (
	defaultApiVersion  = "v1beta1"
	defaultInsecure    = false
	defaultKubeletPort = 10255
)

func init() {
	extpoints.SourceFactories.Register(CreateKubeSources, "kubernetes")
}

func CreateKubeSources(uri string, options map[string][]string) ([]api.Source, error) {
	parsedUrl, err := url.Parse(os.ExpandEnv(uri))
	if err != nil {
		return nil, err
	}

	if len(parsedUrl.Scheme) == 0 {
		return nil, fmt.Errorf("Missing scheme in Kubernetes source: %v", uri)
	}
	if len(parsedUrl.Host) == 0 {
		return nil, fmt.Errorf("Missing host in Kubernetes source: %v", uri)
	}

	kubeConfig := kube_client.Config{
		Host:     fmt.Sprintf("%s://%s", parsedUrl.Scheme, parsedUrl.Host),
		Version:  defaultApiVersion,
		Insecure: defaultInsecure,
	}
	if len(options["apiVersion"]) >= 1 {
		kubeConfig.Version = options["apiVersion"][0]
	}

	if len(options["insecure"]) > 0 {
		insecure, err := strconv.ParseBool(options["insecure"][0])
		if err != nil {
			return nil, err
		}
		kubeConfig.Insecure = insecure
	}

	if len(options["auth"]) > 0 {
		clientAuth, err := clientauth.LoadFromFile(options["auth"][0])
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
	kubeletPort := defaultKubeletPort
	if len(options["kubeletPort"]) >= 1 {
		kubeletPort, err = strconv.Atoi(options["kubeletPort"][0])
		if err != nil {
			return nil, err
		}
	}
	glog.Infof("Using Kubernetes client with master %q and version %s\n", kubeConfig.Host, kubeConfig.Version)
	glog.Infof("Using kubelet port %d", kubeletPort)
	kubeletApi := datasource.NewKubelet()
	kubePodsSource := NewKubePodMetrics(kubeletPort, nodesApi, newPodsApi(kubeClient), kubeletApi)
	kubeNodeSource := NewKubeNodeMetrics(kubeletPort, kubeletApi, nodesApi)
	kubeEventsSource := NewKubeEvents(kubeClient)

	return []api.Source{kubePodsSource, kubeNodeSource, kubeEventsSource}, nil
}
