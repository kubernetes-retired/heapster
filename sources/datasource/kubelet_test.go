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

package datasource

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	cadvisor_api "github.com/google/cadvisor/info/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkContainer(t *testing.T, expected api.Container, actual api.Container) {
	assert.True(t, actual.Spec.Eq(&expected.Spec))
	for i, stat := range actual.Stats {
		assert.True(t, stat.Eq(expected.Stats[i]))
	}
}

func TestBasicKubelet(t *testing.T) {
	response := cadvisor_api.ContainerInfo{}
	data, err := json.Marshal(&response)
	require.NoError(t, err)
	handler := util.FakeHandler{
		StatusCode:   200,
		RequestBody:  "",
		ResponseBody: string(data),
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()
	kubeletSource := kubeletSource{}
	container, err := kubeletSource.getContainer(server.URL, time.Now(), time.Now().Add(time.Minute), time.Second, false)
	require.NoError(t, err)
	assert.Nil(t, container)
}

func TestDetailedKubelet(t *testing.T) {
	rootContainer := api.Container{
		Name: "/",
		Spec: cadvisor_api.ContainerSpec{
			CreationTime: time.Now(),
			HasCpu:       true,
			HasMemory:    true,
		},
		Stats: []*cadvisor_api.ContainerStats{
			{
				Timestamp: time.Now(),
			},
		},
	}
	response := cadvisor_api.ContainerInfo{
		ContainerReference: cadvisor_api.ContainerReference{
			Name: rootContainer.Name,
		},
		Spec:  rootContainer.Spec,
		Stats: rootContainer.Stats,
	}
	data, err := json.Marshal(&response)
	require.NoError(t, err)
	handler := util.FakeHandler{
		StatusCode:   200,
		RequestBody:  "",
		ResponseBody: string(data),
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()
	kubeletSource := kubeletSource{}
	container, err := kubeletSource.getContainer(server.URL, time.Now(), time.Now().Add(time.Minute), time.Second, false)
	require.NoError(t, err)
	checkContainer(t, rootContainer, *container)
}

func TestAllContainers(t *testing.T) {
	rootContainer := api.Container{
		Name: "/",
		Spec: cadvisor_api.ContainerSpec{
			CreationTime: time.Now(),
			HasCpu:       true,
			HasMemory:    true,
		},
		Stats: []*cadvisor_api.ContainerStats{
			{
				Timestamp: time.Now(),
			},
		},
	}
	subcontainer := api.Container{
		Name: "/sub",
		Spec: cadvisor_api.ContainerSpec{
			CreationTime: time.Now(),
			HasCpu:       true,
			HasMemory:    true,
		},
		Stats: []*cadvisor_api.ContainerStats{
			{
				Timestamp: time.Now(),
			},
		},
	}
	response := map[string]cadvisor_api.ContainerInfo{
		rootContainer.Name: {
			ContainerReference: cadvisor_api.ContainerReference{
				Name: rootContainer.Name,
			},
			Spec:  rootContainer.Spec,
			Stats: rootContainer.Stats,
		},
		subcontainer.Name: {
			ContainerReference: cadvisor_api.ContainerReference{
				Name: subcontainer.Name,
			},
			Spec:  subcontainer.Spec,
			Stats: subcontainer.Stats,
		},
	}
	data, err := json.Marshal(&response)
	require.NoError(t, err)
	handler := util.FakeHandler{
		StatusCode:   200,
		RequestBody:  "",
		ResponseBody: string(data),
		T:            t,
	}
	server := httptest.NewServer(&handler)
	defer server.Close()
	kubeletSource := kubeletSource{}
	containers, err := kubeletSource.getAllContainers(server.URL, time.Now(), time.Now().Add(time.Minute), time.Second, false)
	require.NoError(t, err)
	require.Len(t, containers, 2)
	checkContainer(t, rootContainer, containers[0])
	checkContainer(t, subcontainer, containers[1])
}
