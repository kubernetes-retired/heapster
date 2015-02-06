// Copyright 2014 Google Inc. All Rights Reserved.
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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/testapi"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/stretchr/testify/require"
)

type fakeNodesApi struct {
	nodeList nodes.NodeList
}

func (self *fakeNodesApi) List() (*nodes.NodeList, error) {
	return &self.nodeList, nil
}

func (self *fakeNodesApi) DebugInfo() string {
	return ""
}

type fakePodsApi struct {
	podList []Pod
}

func (self *fakePodsApi) List(nodeList *nodes.NodeList) ([]Pod, error) {
	return self.podList, nil
}

func (self *fakePodsApi) DebugInfo() string {
	return ""
}

func TestKubeSourceBasic(t *testing.T) {
	handler := util.FakeHandler{
		StatusCode:   200,
		RequestBody:  "",
		ResponseBody: "",
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()
	client := client.NewOrDie(&client.Config{Host: server.URL, Version: testapi.Version()})
	nodesApi := &fakeNodesApi{nodes.NodeList{}}
	podsApi := &fakePodsApi{[]Pod{}}
	kubeSource := &kubeSource{
		client:      client,
		lastQuery:   time.Now(),
		kubeletPort: "10250",
		nodesApi:    nodesApi,
		podsApi:     podsApi,
	}
	_, err := kubeSource.GetInfo()
	require.NoError(t, err)
	require.NotEmpty(t, kubeSource.DebugInfo())
}
