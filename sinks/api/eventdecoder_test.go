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
	"testing"

	source_api "github.com/GoogleCloudPlatform/heapster/sources/api"
	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kube_time "github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestEventDecoderEmptyInput(t *testing.T) {
	// Arrange
	events := []kube_api.Event{}
	input := source_api.AggregateData{
		Events: events,
	}

	// Act
	timeseries, err := NewEventDecoder().Timeseries(input)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, timeseries)
}

func TestEventDecoderSingleEvent(t *testing.T) {
	// Arrange
	eventTime := kube_time.Unix(12345, 0)
	eventSourceHostname := "event1HostName"
	events := []kube_api.Event{
		kube_api.Event{
			Reason:        "event1",
			LastTimestamp: eventTime,
			Source: kube_api.EventSource{
				Host: eventSourceHostname,
			},
		},
	}
	input := source_api.AggregateData{
		Events: events,
	}

	// Act
	timeseries, err := NewEventDecoder().Timeseries(input)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	assert.Equal(t, 1, len(timeseries))
	assert.Equal(t, EventsSeriesName, timeseries[0].SeriesName)
	assert.Equal(t, Millisecond, timeseries[0].TimePrecision)
	assert.Equal(t, SupportedLabelKeys(), timeseries[0].LabelKeys)
	assert.Equal(t, 1, len(timeseries[0].Points))
	assert.Equal(t, eventTime.Time, timeseries[0].Points[0].Start)
	assert.Equal(t, eventTime.Time, timeseries[0].Points[0].End)
	assert.Equal(t, 1, len(timeseries[0].Points[0].Labels))
	assert.Equal(t, eventSourceHostname, timeseries[0].Points[0].Labels[labelHostname])
	expectedValue := "{\n \"metadata\": {\n  \"creationTimestamp\": null\n },\n \"involvedObject\": {},\n \"reason\": \"event1\",\n \"source\": {\n  \"host\": \"event1HostName\"\n },\n \"firstTimestamp\": null,\n \"lastTimestamp\": \"1970-01-01T03:25:45Z\"\n}"
	assert.Equal(t, expectedValue, timeseries[0].Points[0].Value)
}

func TestEventDecoderMultipleEvents(t *testing.T) {
	// Arrange
	event1Time := kube_time.Unix(12345, 0)
	event2Time := kube_time.Unix(54321, 0)
	event1SourceHostname := "event1HostName"
	event2SourceHostname := "event2HostName"
	events := []kube_api.Event{
		kube_api.Event{
			Reason:        "event1",
			LastTimestamp: event1Time,
			Source: kube_api.EventSource{
				Host: event1SourceHostname,
			},
		},
		kube_api.Event{
			Reason:        "event2",
			LastTimestamp: event2Time,
			Source: kube_api.EventSource{
				Host: event2SourceHostname,
			},
		},
	}
	input := source_api.AggregateData{
		Events: events,
	}

	// Act
	timeseries, err := NewEventDecoder().Timeseries(input)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, timeseries)
	assert.Equal(t, 1, len(timeseries))
	assert.Equal(t, EventsSeriesName, timeseries[0].SeriesName)
	assert.Equal(t, Millisecond, timeseries[0].TimePrecision)
	assert.Equal(t, SupportedLabelKeys(), timeseries[0].LabelKeys)
	assert.Equal(t, 2, len(timeseries[0].Points))
	assert.Equal(t, event1Time.Time, timeseries[0].Points[0].Start)
	assert.Equal(t, event1Time.Time, timeseries[0].Points[0].End)
	assert.Equal(t, 1, len(timeseries[0].Points[0].Labels))
	assert.Equal(t, event1SourceHostname, timeseries[0].Points[0].Labels[labelHostname])
	expectedValue := "{\n \"metadata\": {\n  \"creationTimestamp\": null\n },\n \"involvedObject\": {},\n \"reason\": \"event1\",\n \"source\": {\n  \"host\": \"event1HostName\"\n },\n \"firstTimestamp\": null,\n \"lastTimestamp\": \"1970-01-01T03:25:45Z\"\n}"
	assert.Equal(t, expectedValue, timeseries[0].Points[0].Value)
	assert.Equal(t, event2Time.Time, timeseries[0].Points[1].Start)
	assert.Equal(t, event2Time.Time, timeseries[0].Points[1].End)
	assert.Equal(t, 1, len(timeseries[0].Points[1].Labels))
	assert.Equal(t, event2SourceHostname, timeseries[0].Points[1].Labels[labelHostname])
	expectedValue = "{\n \"metadata\": {\n  \"creationTimestamp\": null\n },\n \"involvedObject\": {},\n \"reason\": \"event2\",\n \"source\": {\n  \"host\": \"event2HostName\"\n },\n \"firstTimestamp\": null,\n \"lastTimestamp\": \"1970-01-01T15:05:21Z\"\n}"
	assert.Equal(t, expectedValue, timeseries[0].Points[1].Value)

}
