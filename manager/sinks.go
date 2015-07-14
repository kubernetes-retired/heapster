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

package manager

import (
	"fmt"
	"net/url"

	"github.com/GoogleCloudPlatform/heapster/extpoints"
	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
)

func newSinks(sinkUris []string) ([]sink_api.ExternalSink, error) {
	var sinks []sink_api.ExternalSink
	for _, sinkFlag := range sinkUris {
		uri, err := url.Parse(sinkFlag)
		if err != nil {
			return nil, err
		}
		if (uri.Scheme == "" || uri.Opaque == "") && uri.Path == "" {
			return nil, fmt.Errorf("Invalid sink definition: %s", sinkFlag)
		}
		key := uri.Scheme
		if key == "" {
			key = uri.Path
		}
		factory := extpoints.SinkFactories.Lookup(key)
		if factory == nil {
			return nil, fmt.Errorf("Unknown sink: %s", key)
		}
		createdSinks, err := factory(uri.Opaque, uri.Query())
		if err != nil {
			return nil, err
		}
		sinks = append(sinks, createdSinks...)
	}
	return sinks, nil
}
