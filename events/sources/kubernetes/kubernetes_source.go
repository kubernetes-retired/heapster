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

package kubernetes

import (
	"net/url"
	"time"

	"github.com/golang/glog"

	kubeconfig "k8s.io/heapster/common/kubernetes"
	"k8s.io/heapster/events/core"
	kubeapi "k8s.io/kubernetes/pkg/api"
	kubeapiunv "k8s.io/kubernetes/pkg/api/unversioned"
	kubeclient "k8s.io/kubernetes/pkg/client/unversioned"
	kubefields "k8s.io/kubernetes/pkg/fields"
	kubelabels "k8s.io/kubernetes/pkg/labels"
	kubewatch "k8s.io/kubernetes/pkg/watch"
)

const (
	// Number of object pointers. Big enough so it won't be hit anytime soon with resonable GetNewEvents frequency.
	LocalEventsBufferSize = 100000
)

// Implements core.EventSource interface.
type KubernetesEventSource struct {
	// Large local buffer, periodically read.
	localEventsBuffer chan *kubeapi.Event

	stopChannel chan struct{}

	eventClient kubeclient.EventInterface
}

func (this *KubernetesEventSource) GetNewEvents() *core.EventBatch {
	result := core.EventBatch{
		Timestamp: time.Now(),
		Events:    []*kubeapi.Event{},
	}
	// Get all data from the buffer.
event_loop:
	for {
		select {
		case event := <-this.localEventsBuffer:
			result.Events = append(result.Events, event)
		default:
			break event_loop
		}
	}
	return &result
}

func (this *KubernetesEventSource) watch() {
	// Outer loop, for reconnections.
	for {
		events, err := this.eventClient.List(kubelabels.Everything(), kubefields.Everything())
		if err != nil {
			glog.Errorf("Failed to load events: %v", err)
			time.Sleep(time.Second)
			continue
		}
		// Do not write old events.

		resourceVersion := events.ResourceVersion

		watcher, err := this.eventClient.Watch(
			kubelabels.Everything(),
			kubefields.Everything(),
			kubeapi.ListOptions{
				LabelSelector:   kubelabels.Everything(),
				FieldSelector:   kubefields.Everything(),
				Watch:           true,
				ResourceVersion: resourceVersion})
		if err != nil {
			glog.Errorf("Failed to start watch for new events: %v", err)
			time.Sleep(time.Second)
			continue
		}

		watchChannel := watcher.ResultChan()
		// Inner loop, for update processing.
	inner_loop:
		for {
			select {
			case watchUpdate, ok := <-watchChannel:
				if !ok {
					glog.Errorf("Event watch channel closed")
					break inner_loop
				}

				if watchUpdate.Type == kubewatch.Error {
					if status, ok := watchUpdate.Object.(*kubeapiunv.Status); ok {
						glog.Errorf("Error during watch: %#v", status)
						break inner_loop
					}
					glog.Errorf("Received unexpected error: %#v", watchUpdate.Object)
					break inner_loop
				}

				if event, ok := watchUpdate.Object.(*kubeapi.Event); ok {
					switch watchUpdate.Type {
					case kubewatch.Added, kubewatch.Modified:
						select {
						case this.localEventsBuffer <- event:
							// Ok, buffer not full.
						default:
							// Buffer full, need to drop the event.
							glog.Errorf("Event buffer full, dropping event")
						}
					case kubewatch.Deleted:
						// Deleted events are silently ignored.
					default:
						glog.Warningf("Unknown watchUpdate.Type: %#v", watchUpdate.Type)
					}
				} else {
					glog.Errorf("Wrong object received: %v", watchUpdate)
				}

			case <-this.stopChannel:
				glog.Infof("Event watching stopped")
				return
			}
		}
	}
}

func NewKubernetesSource(uri *url.URL) (*KubernetesEventSource, error) {
	kubeConfig, err := kubeconfig.GetKubeClientConfig(uri)
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubeclient.New(kubeConfig)
	if err != nil {
		return nil, err
	}
	eventClient := kubeClient.Events(kubeapi.NamespaceAll)
	result := KubernetesEventSource{
		localEventsBuffer: make(chan *kubeapi.Event, LocalEventsBufferSize),
		stopChannel:       make(chan struct{}),
		eventClient:       eventClient,
	}
	go result.watch()
	return &result, nil
}
