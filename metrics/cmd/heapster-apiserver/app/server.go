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

// Package app does all of the work necessary to create a Heapster
// APIServer by binding together the Master Metrics API.
// It can be configured and called directly or via the hyperkube framework.
package app

import (
	"fmt"

	"github.com/pborman/uuid"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/heapster/metrics/options"
	"k8s.io/heapster/metrics/sinks/metric"
	genericapiserver "k8s.io/kubernetes/pkg/genericapiserver/server"
)

type HeapsterAPIServer struct {
	*genericapiserver.GenericAPIServer
	options    *options.HeapsterRunOptions
	metricSink *metricsink.MetricSink
	nodeLister *cache.StoreToNodeLister
}

// Run runs the specified APIServer. This should never exit.
func (h *HeapsterAPIServer) RunServer() error {
	h.PrepareRun().Run(wait.NeverStop)
	return nil
}

func NewHeapsterApiServer(s *options.HeapsterRunOptions, metricSink *metricsink.MetricSink,
	nodeLister *cache.StoreToNodeLister, podLister *cache.StoreToPodLister) (*HeapsterAPIServer, error) {

	server, err := newAPIServer(s)
	if err != nil {
		return &HeapsterAPIServer{}, err
	}

	installMetricsAPIs(s, server, metricSink, nodeLister, podLister)

	return &HeapsterAPIServer{
		GenericAPIServer: server,
		options:          s,
		metricSink:       metricSink,
		nodeLister:       nodeLister,
	}, nil
}

func newAPIServer(s *options.HeapsterRunOptions) (*genericapiserver.GenericAPIServer, error) {
	if err := s.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost"); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	genericAPIServerConfig := genericapiserver.NewConfig()
	if _, err := genericAPIServerConfig.ApplySecureServingOptions(s.SecureServing); err != nil {
		return nil, err
	}

	if !s.DisableAuthForTesting {
		if _, err := genericAPIServerConfig.ApplyDelegatingAuthenticationOptions(s.Authentication); err != nil {
			return nil, err
		}
		if _, err := genericAPIServerConfig.ApplyDelegatingAuthorizationOptions(s.Authorization); err != nil {
			return nil, err
		}
	}

	var err error
	privilegedLoopbackToken := uuid.NewRandom().String()
	if genericAPIServerConfig.LoopbackClientConfig, err = genericAPIServerConfig.SecureServingInfo.NewSelfClientConfig(privilegedLoopbackToken); err != nil {
		return nil, err
	}

	genericAPIServerConfig.SwaggerConfig = genericapiserver.DefaultSwaggerConfig()

	// TODO: Include EnableWatchCache and other GenericServerRunOptions
	// TODO: AdvertiseAddress?

	return genericAPIServerConfig.Complete().New()
}
