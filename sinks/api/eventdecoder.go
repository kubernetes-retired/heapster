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

package api

import (
	"encoding/json"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

const (
	EventsSeriesName = "log/events"
)

type eventDecoder struct{}

func NewEventDecoder() Decoder {
	return &eventDecoder{}
}

func (self *eventDecoder) Timeseries(data source_api.AggregateData) ([]Timeseries, error) {
	result := []Timeseries{}
	if data.Events != nil && len(data.Events) > 0 {
		points := []Point{}
		for _, event := range data.Events {
			value, err := getEventValue(&event)
			if err != nil {
				return result, err
			}
			points = append(points, Point{
				Labels: getEventLabels(&event),
				Start:  event.LastTimestamp.Time,
				End:    event.LastTimestamp.Time,
				Value:  value,
			})
		}
		result = append(result, Timeseries{
			SeriesName:    EventsSeriesName,
			TimePrecision: Millisecond,
			LabelKeys:     SupportedLabelKeys(),
			Points:        points,
		})
	}

	return result, nil

}

// Generate the labels.
func getEventLabels(event *kube_api.Event) map[string]string {
	labels := make(map[string]string)
	if event == nil {
		return nil
	}

	labels[labelHostname] = event.Source.Host
	if event.InvolvedObject.Kind == "Pod" {
		labels[labelPodName] = event.InvolvedObject.Name
		labels[labelPodId] = string(event.InvolvedObject.UID)
	}

	return labels
}

// Generate point value for event
func getEventValue(event *kube_api.Event) (string, error) {
	bytes, err := json.MarshalIndent(event, "", " ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
