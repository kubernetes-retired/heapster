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
	"errors"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/api"
	kubeapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kubeclient "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kubefields "github.com/GoogleCloudPlatform/kubernetes/pkg/fields"
	kubelabels "github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	kubewatch "github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
	"github.com/golang/glog"
)

const kubeEventsSource = "kube-events"

// eventsUpdate is the wrapper object used to pass new events around
type eventsUpdate struct {
	events *kubeapi.EventList
}

// eventsSourceImpl is an implmentation of eventsSource
type eventsSourceImpl struct {
	*kubeclient.Client
	eventsChannel chan eventsUpdate
	errorChannel  chan error
	initialized   bool
}

// Terminates existing watch loop, if any, and starts new instance
// Note that the current implementation will cause all events that
// haven't been removed due TTL to be redelivered.
func (eventSource *eventsSourceImpl) restartWatchLoop() {
	eventSource.eventsChannel = make(chan eventsUpdate, 1024)
	eventSource.errorChannel = make(chan error)
	glog.V(4).Infof("Restarting event source")
	go watchLoop(eventSource.Client.Events(kubeapi.NamespaceAll), eventSource.eventsChannel, eventSource.errorChannel)
	glog.V(4).Infof("Finished restarting event source")
}

// getEvents returns all new events since getEvents was last called.
func (eventSource *eventsSourceImpl) getEvents() ([]kubeapi.Event, bool, error) {
	events := []kubeapi.Event{}
UpdateLoop:
	for {
		// Non-blocking receive
		select {
		case eventsUpdate, ok := <-eventSource.eventsChannel:
			if !ok {
				return nil, true, fmt.Errorf("eventsChannel was closed")
			}
			if eventsUpdate.events == nil {
				return nil, false, fmt.Errorf("Error: recieved a nil event list.")
			}
			if eventsUpdate.events.Items == nil {
				return nil, false, fmt.Errorf("Error: received an event list with nil Items.")
			}
			for _, event := range eventsUpdate.events.Items {
				glog.V(3).Infof("Received new event: %#v\r\n", event)
				events = append(events, event)
			}
		case err := <-eventSource.errorChannel:
			if err != nil {
				err = fmt.Errorf("Events watchLoop failed with error: %v", err)
				return nil, true, err
			}
		default:
			break UpdateLoop
		}
	}
	return events, false, nil
}

// watchLoop loops forever looking for new events.  If an error occurs it will close the channel and return.
func watchLoop(eventClient kubeclient.EventInterface, eventsChan chan<- eventsUpdate, errorChan chan<- error) {
	defer close(eventsChan)
	defer close(errorChan)
	events, err := eventClient.List(kubelabels.Everything(), kubefields.Everything())
	if err != nil {
		glog.Errorf("Failed to load events: %v", err)
		errorChan <- err
		return
	}
	resourceVersion := events.ResourceVersion
	eventsChan <- eventsUpdate{events: events}

	watcher, err := eventClient.Watch(kubelabels.Everything(), kubefields.Everything(), resourceVersion)
	if err != nil {
		glog.Errorf("Failed to start watch for new events: %v", err)
		errorChan <- err
		return
	}
	defer watcher.Stop()

	watchChannel := watcher.ResultChan()
	for {
		watchUpdate, ok := <-watchChannel
		if !ok {
			err := errors.New("watchLoop channel closed")
			errorChan <- err
			return
		}

		if watchUpdate.Type == kubewatch.Error {
			if status, ok := watchUpdate.Object.(*kubeapi.Status); ok {
				err := fmt.Errorf("Error during watch: %#v", status)
				errorChan <- err
				return
			}
			err := fmt.Errorf("Received unexpected error: %#v", watchUpdate.Object)
			errorChan <- err
			return
		}

		if event, ok := watchUpdate.Object.(*kubeapi.Event); ok {

			switch watchUpdate.Type {
			case kubewatch.Added, kubewatch.Modified:
				eventsChan <- eventsUpdate{&kubeapi.EventList{Items: []kubeapi.Event{*event}}}
			case kubewatch.Deleted:
				// Deleted events are silently ignored
			default:
				err := fmt.Errorf("Unknown watchUpdate.Type: %#v", watchUpdate.Type)
				errorChan <- err
				return
			}
			resourceVersion = event.ResourceVersion
			continue
		}
	}
}

func NewKubeEvents(client *kubeclient.Client) api.Source {
	// Buffered channel to send/receive events from
	eventsChan := make(chan eventsUpdate, 1024)
	errorChan := make(chan error)
	glog.V(4).Infof("Starting %q source", kubeEventsSource)
	go watchLoop(client.Events(kubeapi.NamespaceAll), eventsChan, errorChan)
	glog.V(4).Infof("Finished starting %q source", kubeEventsSource)

	return &eventsSourceImpl{
		Client:        client,
		eventsChannel: eventsChan,
		errorChannel:  errorChan,
	}
}

func (eventSource *eventsSourceImpl) GetInfo(start, end time.Time, resolution time.Duration) (api.AggregateData, error) {
	events, watchLoopTerminated, err := eventSource.getEvents()
	if err != nil {
		if watchLoopTerminated {
			glog.Errorf("Event watch loop was terminated due to error. Will restart it. Error: %v", err)
			eventSource.restartWatchLoop()
		}
		return api.AggregateData{}, err
	}
	glog.V(2).Info("Fetched list of events from the master")
	glog.V(4).Infof("%v", events)

	return api.AggregateData{Events: events}, nil
}

func (eventSource *eventsSourceImpl) DebugInfo() string {
	desc := fmt.Sprintf("Source type: %s\n", kubeEventsSource)
	// TODO: Add events specific debug information
	return desc
}
