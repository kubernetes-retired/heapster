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

	"github.com/GoogleCloudPlatform/heapster/sources/nodes"
	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/latest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/testapi"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func body(obj runtime.Object) string {
	if obj != nil {
		bs, _ := latest.Codec.Encode(obj)
		body := string(bs)
		return body
	}

	return ""
}

func TestPodsApiCreation(t *testing.T) {
	handler := util.FakeHandler{
		StatusCode:   200,
		RequestBody:  "something",
		ResponseBody: body(&kube_api.PodList{}),
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()
	client := client.NewOrDie(&client.Config{Host: server.URL, Version: testapi.Version()})
	podsApi := newPodsApi(client)
	_, err := podsApi.List(&nodes.NodeList{})
	require.NoError(t, err)
}

func TestPodsParsing(t *testing.T) {
	podList := &kube_api.PodList{
		Items: []kube_api.Pod{
			{
				Status: kube_api.PodStatus{
					Phase:  kube_api.PodRunning,
					PodIP:  "1.2.3.4",
					Host:   "test-machine-a",
					HostIP: "10.10.10.0",
				},
				ObjectMeta: kube_api.ObjectMeta{
					Name:      "pod1",
					Namespace: "test",
					UID:       "x-y-z",
					Labels: map[string]string{
						"foo":  "bar",
						"name": "baz",
					},
				},
				Spec: kube_api.PodSpec{
					Containers: []kube_api.Container{
						{Name: "test1"},
						{Name: "test2"},
					},
				},
			},
			{
				Status: kube_api.PodStatus{
					Phase:  kube_api.PodRunning,
					PodIP:  "1.2.3.5",
					Host:   "test-machine-b",
					HostIP: "10.10.10.1",
				},
				ObjectMeta: kube_api.ObjectMeta{
					Name:      "pod2",
					Namespace: "test",
					UID:       "x-y-a",
					Labels: map[string]string{
						"foo":  "bar",
						"name": "baz",
					},
				},
				Spec: kube_api.PodSpec{
					Containers: []kube_api.Container{
						{Name: "test1"},
						{Name: "test2"},
					},
				},
			},
		},
	}
	handler := util.FakeHandler{
		StatusCode:   200,
		RequestBody:  "something",
		ResponseBody: body(podList),
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()
	client := client.NewOrDie(&client.Config{Host: server.URL, Version: testapi.Version()})
	podsApi := newPodsApi(client)
	nodeList := &nodes.NodeList{
		Items: map[nodes.Host]nodes.Info{
			nodes.Host("test-machine-b"): {PublicIP: "10.10.10.1"},
			nodes.Host("test-machine-1"): {PublicIP: "10.10.10.0"},
		},
	}
	pods, err := podsApi.List(nodeList)
	require.NoError(t, err)
	for i, pod := range pods {
		assert.Equal(t, pod.Name, podList.Items[i].Name)
		assert.Equal(t, pod.Namespace, podList.Items[i].Namespace)
		assert.Equal(t, pod.ID, podList.Items[i].UID)
		assert.Equal(t, pod.Hostname, podList.Items[i].Status.Host)
		assert.Equal(t, pod.PodIP, podList.Items[i].Status.PodIP)
		assert.Equal(t, pod.Status, podList.Items[i].Status.Phase)
		assert.Equal(t, pod.Labels, podList.Items[i].Labels)
		for idx, container := range pod.Containers {
			assert.Equal(t, container.Name, podList.Items[i].Spec.Containers[idx].Name)
		}
	}
}
