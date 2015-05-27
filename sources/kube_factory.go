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
	kubeClientCmd "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	kubeClientCmdApi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/golang/glog"
)

const (
	defaultApiVersion     = "v1beta3"
	defaultInsecure       = false
	defaultKubeletPort    = 10255
	defaultKubeConfigFile = "/etc/kubernetes/kubeconfig/kubeconfig"
)

func init() {
	extpoints.SourceFactories.Register(CreateKubeSources, "kubernetes")
}

func getConfigOverrides(uri string, options map[string][]string) (*kubeClientCmd.ConfigOverrides, error) {
	kubeConfigOverride := kubeClientCmd.ConfigOverrides{
		ClusterInfo: kubeClientCmdApi.Cluster{
			APIVersion: defaultApiVersion,
		},
	}
	if uri != "" {
		parsedUrl, err := url.Parse(os.ExpandEnv(uri))
		if err != nil {
			return nil, err
		}
		if len(parsedUrl.Scheme) != 0 && len(parsedUrl.Host) != 0 {
			kubeConfigOverride.ClusterInfo.Server = fmt.Sprintf("%s://%s", parsedUrl.Scheme, parsedUrl.Host)
		}
	}

	if len(options["apiVersion"]) >= 1 {
		kubeConfigOverride.ClusterInfo.APIVersion = options["apiVersion"][0]
	}

	if len(options["insecure"]) > 0 {
		insecure, err := strconv.ParseBool(options["insecure"][0])
		if err != nil {
			return nil, err
		}
		kubeConfigOverride.ClusterInfo.InsecureSkipTLSVerify = insecure
	}

	return &kubeConfigOverride, nil
}

func CreateKubeSources(uri string, options map[string][]string) ([]api.Source, error) {
	configOverrides, err := getConfigOverrides(uri, options)

	authFile := defaultKubeConfigFile
	if len(options["auth"]) > 0 {
		authFile = options["auth"][0]
	}

	var kubeConfig *kube_client.Config
	if authFile != "" {
		if kubeConfig, err = kubeClientCmd.NewNonInteractiveDeferredLoadingClientConfig(
			&kubeClientCmd.ClientConfigLoadingRules{ExplicitPath: authFile},
			configOverrides).ClientConfig(); err != nil {
			return nil, err
		}
	} else {
		kubeConfig = &kube_client.Config{
			Host:     configOverrides.ClusterInfo.Server,
			Version:  configOverrides.ClusterInfo.APIVersion,
			Insecure: configOverrides.ClusterInfo.InsecureSkipTLSVerify,
		}
	}
	if len(kubeConfig.Host) == 0 {
		return nil, fmt.Errorf("invalid kubernetes master url specified")
	}
	if len(kubeConfig.Version) == 0 {
		return nil, fmt.Errorf("invalid kubernetes API version specified")
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
	glog.Infof("Using Kubernetes client with master %q and version %q\n", kubeConfig.Host, kubeConfig.Version)
	glog.Infof("Using kubelet port %d", kubeletPort)
	kubeletApi := datasource.NewKubelet()
	kubePodsSource := NewKubePodMetrics(kubeletPort, nodesApi, newPodsApi(kubeClient), kubeletApi)
	kubeNodeSource := NewKubeNodeMetrics(kubeletPort, kubeletApi, nodesApi)
	kubeEventsSource := NewKubeEvents(kubeClient)

	return []api.Source{kubePodsSource, kubeNodeSource, kubeEventsSource}, nil
}
