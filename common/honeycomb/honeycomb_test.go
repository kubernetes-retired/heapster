// Copyright 2017 Google Inc. All Rights Reserved.
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

package honeycomb

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	util "k8s.io/client-go/util/testing"
)

func TestHoneycombClientWrite(t *testing.T) {
	handler := util.FakeHandler{
		StatusCode:   202,
		ResponseBody: "",
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()

	stubURL, err := url.Parse("?writekey=testkey&dataset=testdataset&apihost=" + server.URL)

	assert.NoError(t, err)

	config, err := BuildConfig(stubURL)

	assert.Equal(t, config.WriteKey, "testkey")
	assert.Equal(t, config.APIHost, server.URL)
	assert.Equal(t, config.Dataset, "testdataset")

	assert.NoError(t, err)

	client, _ := NewClient(stubURL)

	err = client.SendBatch([]*BatchPoint{
		{
			Data:      "test",
			Timestamp: time.Now(),
		},
	})

	assert.NoError(t, err)

	handler.ValidateRequestCount(t, 1)
}
