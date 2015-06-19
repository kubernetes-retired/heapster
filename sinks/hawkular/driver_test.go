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

package hawkular

import (
	"fmt"
	"testing"
	"time"

	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
	"github.com/hawkular/hawkular-client-go/metrics"

	assert "github.com/stretchr/testify/require"
)

func dummySink() *hawkularSink {
	return &hawkularSink{
		reg:    make(map[string]*metrics.MetricDefinition),
		models: make(map[string]metrics.MetricDefinition),
	}
}

func TestDescriptorTransform(t *testing.T) {

	hSink := dummySink()

	ld := sink_api.LabelDescriptor{
		Key:         "k1",
		Description: "d1",
	}
	smd := sink_api.MetricDescriptor{
		Name:      "test/metric/1",
		Units:     sink_api.UnitsBytes,
		ValueType: sink_api.ValueInt64,
		Type:      sink_api.MetricGauge,
		Labels:    []sink_api.LabelDescriptor{ld},
	}

	md := hSink.descriptorToDefinition(&smd)

	assert.Equal(t, smd.Name, md.Id)
	assert.Equal(t, 4, len(md.Tags)) // descriptorTag, unitsTag, typesTag, k1

	assert.Equal(t, smd.Units.String(), md.Tags[unitsTag])
	assert.Equal(t, "d1", md.Tags["k1_description"])
}

func TestMetricTransform(t *testing.T) {
	hSink := dummySink()

	smd := sink_api.MetricDescriptor{
		ValueType: sink_api.ValueInt64,
		Type:      sink_api.MetricCumulative,
	}

	l := make(map[string]string)
	l["spooky"] = "notvisible"
	l[sink_api.LabelHostname.Key] = "localhost"
	l[sink_api.LabelContainerName.Key] = "docker"
	l[sink_api.LabelPodId.Key] = "aaaa-bbbb-cccc-dddd"

	p := sink_api.Point{
		Name:   "test/metric/1",
		Labels: l,
		Start:  time.Now(),
		End:    time.Now(),
		Value:  int64(123456),
	}

	ts := sink_api.Timeseries{
		MetricDescriptor: &smd,
		Point:            &p,
	}

	m, err := hSink.pointToMetricHeader(&ts)
	assert.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("%s/%s/%s", p.Labels[sink_api.LabelContainerName.Key], p.Labels[sink_api.LabelPodId.Key], p.Name), m.Id)

	assert.Equal(t, 1, len(m.Data))
	_, ok := m.Data[0].Value.(float64)
	assert.True(t, ok, "Value should have been converted to float64")
}

func TestRecentTest(t *testing.T) {
	hSink := dummySink()

	modelT := make(map[string]string)

	id := "test.name"
	modelT[descriptorTag] = "d"
	modelT[groupTag] = id
	modelT["hep"+descriptionTag] = "n"

	model := metrics.MetricDefinition{
		Id:   id,
		Tags: modelT,
	}

	liveT := make(map[string]string)
	for k, v := range modelT {
		liveT[k] = v
	}

	live := metrics.MetricDefinition{
		Id:   "test/" + id,
		Tags: liveT,
	}

	assert.True(t, hSink.recent(&live, &model), "Tags are equal, live is newest")

	delete(liveT, "hep"+descriptionTag)
	live.Tags = liveT

	assert.False(t, hSink.recent(&live, &model), "Tags are not equal, live isn't recent")

}
