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

	"k8s.io/client-go/tools/cache"
	"k8s.io/heapster/metrics/apis/metrics"
	_ "k8s.io/heapster/metrics/apis/metrics/install"
	"k8s.io/heapster/metrics/apis/metrics/v1alpha1"
	"k8s.io/heapster/metrics/options"
	"k8s.io/heapster/metrics/sinks/metric"
	nodemetricsstorage "k8s.io/heapster/metrics/storage/nodemetrics"
	podmetricsstorage "k8s.io/heapster/metrics/storage/podmetrics"
	"k8s.io/kubernetes/pkg/genericapiserver/registry/rest"
	genericapiserver "k8s.io/kubernetes/pkg/genericapiserver/server"
)

func installMetricsAPIs(s *options.HeapsterRunOptions, g *genericapiserver.GenericAPIServer,
	metricSink *metricsink.MetricSink, nodeLister *cache.StoreToNodeLister, podLister *cache.StoreToPodLister) {

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(metrics.GroupName)

	nodemetricsStorage := nodemetricsstorage.NewStorage(metrics.Resource("nodemetrics"), metricSink, nodeLister)
	podmetricsStorage := podmetricsstorage.NewStorage(metrics.Resource("podmetrics"), metricSink, podLister)
	heapsterResources := map[string]rest.Storage{
		"nodes": nodemetricsStorage,
		"pods":  podmetricsStorage,
	}
	apiGroupInfo.VersionedResourcesStorageMap[v1alpha1.SchemeGroupVersion.Version] = heapsterResources

	if err := g.InstallAPIGroup(&apiGroupInfo); err != nil {
		glog.Fatalf("Error in registering group versions: %v", err)
	}
}
