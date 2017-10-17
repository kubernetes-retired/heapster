// Copyright 2016 Google Inc. All Rights Reserved.
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

package app

import (
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	v1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/heapster/metrics/options"
	metricsink "k8s.io/heapster/metrics/sinks/metric"
	nodemetricsstorage "k8s.io/heapster/metrics/storage/nodemetrics"
	podmetricsstorage "k8s.io/heapster/metrics/storage/podmetrics"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	_ "k8s.io/kubernetes/pkg/apis/core/install"
	"k8s.io/metrics/pkg/apis/metrics"
	metrics_api "k8s.io/metrics/pkg/apis/metrics/v1alpha1"
)

func init() {
	install(legacyscheme.GroupFactoryRegistry, legacyscheme.Registry, legacyscheme.Scheme)
}

func installMetricsAPIs(s *options.HeapsterRunOptions, g *genericapiserver.GenericAPIServer,
	metricSink *metricsink.MetricSink, nodeLister v1listers.NodeLister, podLister v1listers.PodLister) {

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(metrics.GroupName, legacyscheme.Registry, legacyscheme.Scheme, legacyscheme.ParameterCodec, legacyscheme.Codecs)

	nodemetricsStorage := nodemetricsstorage.NewStorage(metrics.Resource("nodemetrics"), metricSink, nodeLister)
	podmetricsStorage := podmetricsstorage.NewStorage(metrics.Resource("podmetrics"), metricSink, podLister)
	heapsterResources := map[string]rest.Storage{
		"nodes": nodemetricsStorage,
		"pods":  podmetricsStorage,
	}
	apiGroupInfo.VersionedResourcesStorageMap[metrics_api.SchemeGroupVersion.Version] = heapsterResources

	if err := g.InstallAPIGroup(&apiGroupInfo); err != nil {
		glog.Fatalf("Error in registering group versions: %v", err)
	}
}

// This function is directly copied from https://github.com/kubernetes/metrics/blob/master/pkg/apis/metrics/install/install.go#L31 with only changes by replacing v1beta1 to v1alpha1.
// This function should be deleted only after move metrics to v1beta1.
// Install registers the API group and adds types to a scheme
func install(groupFactoryRegistry announced.APIGroupFactoryRegistry, registry *registered.APIRegistrationManager, scheme *runtime.Scheme) {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  metrics.GroupName,
			VersionPreferenceOrder:     []string{metrics_api.SchemeGroupVersion.Version},
			RootScopedKinds:            sets.NewString("NodeMetrics"),
			AddInternalObjectsToScheme: metrics.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			metrics_api.SchemeGroupVersion.Version: metrics_api.AddToScheme,
		},
	).Announce(groupFactoryRegistry).RegisterAndEnable(registry, scheme); err != nil {
		panic(err)
	}
}
