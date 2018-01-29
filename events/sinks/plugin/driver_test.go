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

package plugin

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	kube_api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	plugin_common "k8s.io/heapster/common/plugin"
	event_core "k8s.io/heapster/events/core"
)

var (
	errExpected = errors.New("know Error")
)

type testEventSink struct{}

func (sink *testEventSink) WriteEventsPoint(point *PluginSinkPoint) error {
	return errExpected
}

type fakeEventSink struct {
	batchPoints []*PluginSinkPoint
}

func (sink *fakeEventSink) WriteEventsPoint(point *PluginSinkPoint) error {
	sink.batchPoints = append(sink.batchPoints, point)
	return nil
}

func TestGoPluginSinkWritePoint(t *testing.T) {
	uri, err := url.Parse(fmt.Sprintf("plugin:?cmd=%s&cmd=-test.run=TestHelperProcess&cmd=--", os.Args[0]))
	assert.NoError(t, err)
	sink, err := NewGoPluginSink(uri)
	assert.NoError(t, err)
	defer sink.Stop()

	err = sink.(*GoPluginSink).WriteEventsPoint(&PluginSinkPoint{})
	assert.True(t, strings.Contains(err.Error(), errExpected.Error()))
}

// Fake test to be used as a plugin
func TestHelperProcess(t *testing.T) {
	if os.Getenv(plugin_common.Handshake.MagicCookieKey) == "" {
		return
	}

	defer os.Exit(0)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin_common.Handshake,
		Plugins: map[string]plugin.Plugin{
			"events": &EventSinkPlugin{Impl: &testEventSink{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func TestExportEventsEmpty(t *testing.T) {
	fakeSink := &fakeEventSink{}
	sink := &GoPluginSink{
		EventSink: fakeSink,
	}
	eventBatch := event_core.EventBatch{}
	sink.ExportEvents(&eventBatch)
	assert.Equal(t, 0, len(fakeSink.batchPoints))
}

func TestExportEventsContent(t *testing.T) {
	fakeSink := &fakeEventSink{}
	sink := &GoPluginSink{
		EventSink: fakeSink,
	}
	timestamp := time.Now()
	data := event_core.EventBatch{
		Timestamp: timestamp,
		Events: []*kube_api.Event{
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: "abc",
				},
				LastTimestamp: metav1.Time{
					Time: timestamp,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: "123",
				},
				LastTimestamp: metav1.Time{
					Time: timestamp,
				},
			},
		},
	}

	sink.ExportEvents(&data)
	assert.Equal(t, 2, len(fakeSink.batchPoints))
}
