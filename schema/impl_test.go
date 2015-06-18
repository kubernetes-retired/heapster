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

package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAddNamespace tests all flows of addNamespace.
func TestAddNamespace(t *testing.T) {
	cluster := newRealCluster()
	namespace_name := "default"

	// First call : namespace does not exist
	namespace := cluster.addNamespace(namespace_name)

	assert := assert.New(t)
	assert.NotNil(namespace)
	assert.Equal(cluster.Namespaces[namespace_name], namespace)
	assert.NotNil(namespace.Metrics)
	assert.NotNil(namespace.Labels)
	assert.NotNil(namespace.Pods)

	// Second call : namespace already exists
	new_namespace := cluster.addNamespace(namespace_name)
	assert.Equal(new_namespace, namespace)
}

// TestAddNode tests all flows of addNode.
func TestAddNode(t *testing.T) {
	cluster := newRealCluster()
	hostname := "kubernetes-minion-xkhz"

	// First call : node does not exist
	node := cluster.addNode(hostname)

	assert := assert.New(t)
	assert.NotNil(node)
	assert.Equal(cluster.Nodes[hostname], node)
	assert.NotNil(node.Metrics)
	assert.NotNil(node.Labels)
	assert.NotNil(node.FreeContainers)
	assert.NotNil(node.Pods)

	// Second call : node already exists
	new_node := cluster.addNode(hostname)
	assert.Equal(new_node, node)
}

// TestAddPod tests all flows of addPod.
func TestAddPod(t *testing.T) {
	cluster := newRealCluster()
	pod_name := "podname-xkhz"
	pod_uid := "123124-124124-124124124124"
	namespace := cluster.addNamespace("default")
	node := cluster.addNode("kubernetes-minion-xkhz")

	// First call : pod does not exist
	pod := cluster.addPod(pod_name, pod_uid, namespace, node)

	assert := assert.New(t)
	assert.NotNil(pod)
	assert.Equal(node.Pods[pod_name], pod)
	assert.Equal(namespace.Pods[pod_name], pod)
	assert.Equal(pod.UID, pod_uid)
	assert.NotNil(pod.Metrics)
	assert.NotNil(pod.Labels)
	assert.NotNil(pod.Containers)

	// Second call : pod already exists
	new_pod := cluster.addPod(pod_name, pod_uid, namespace, node)
	assert.NotNil(new_pod)
	assert.Equal(new_pod, pod)

	// Third call : References are mismatched
	other_node := cluster.addNode("other-kubernetes-minion")
	newest_pod := cluster.addPod(pod_name, pod_uid, namespace, other_node)
	assert.Nil(newest_pod)
}

// TestUpdateTime tests the sanity of updateTime.
func TestUpdateTime(t *testing.T) {
	cluster := newRealCluster()
	stamp := time.Now()

	assert.NotEqual(t, cluster.timestamp, stamp)
	cluster.updateTime(stamp)
	assert.Equal(t, cluster.timestamp, stamp)
}
