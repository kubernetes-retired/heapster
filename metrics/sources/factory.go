// Copyright 2015 Google Inc. All Rights Reserved.
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

	"k8s.io/heapster/common/flags"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sources/kubelet"
	"k8s.io/heapster/metrics/sources/push"
	"k8s.io/heapster/metrics/sources/summary"
)

type SourceFactory struct {
}

func (this *SourceFactory) Build(uri flags.Uri) (core.MetricsSourceProvider, error) {
	switch uri.Key {
	case "kubernetes":
		provider, err := kubelet.NewKubeletProvider(&uri.Val)
		return provider, err
	case "kubernetes.summary_api":
		provider, err := summary.NewSummaryProvider(&uri.Val)
		return provider, err
	default:
		return nil, fmt.Errorf("Source not recognized: %s", uri.Key)
	}
}

func (this *SourceFactory) BuildAll(uris flags.Uris) (core.MetricsSourceProvider, push.PushSource, error) {
	// We can have 2 sources, if one is a push source
	if len(uris) == 2 {
		// we can't have two push sources
		if uris[0].Key == "push" && uris[1].Key == "push" {
			return nil, nil, fmt.Errorf("Cannot have multiple push sources")
		}

		// if we have more than one source, we need at least one push source
		if uris[0].Key != "push" && uris[1].Key != "push" {
			return nil, nil, fmt.Errorf("Only one non-push source is supported")
		}

		// figure out which source is the "normal" one, and build it
		var provider core.MetricsSourceProvider
		var err error
		if uris[0].Key == "push" {
			provider, err = this.Build(uris[1])
		} else {
			provider, err = this.Build(uris[0])
		}

		if err != nil {
			return nil, nil, err
		}

		return push.NewPushProvider(provider)
	}

	if len(uris) != 1 {
		return nil, nil, fmt.Errorf("Only one non-push source is supported")
	}

	if uris[0].Key == "push" {
		return push.NewPushProvider(nil)
	}

	provider, err := this.Build(uris[0])
	return provider, nil, err
}

func NewSourceFactory() *SourceFactory {
	return &SourceFactory{}
}
