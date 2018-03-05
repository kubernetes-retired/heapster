// Copyright 2018 Google Inc. All Rights Reserved.
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
package metricly

import (
	metricly_core "github.com/metricly/go-client/model/core"
	"k8s.io/heapster/metrics/core"
	"testing"
	"time"
)

func TestConvertDataBatchToElements(t *testing.T) {
	//given
	batch := createDataBatch()
	//when
	elements := DataBatchToElements(batch)
	//then
	if len(elements) != 1 {
		t.Errorf("there should be 1 element in elements, but actual =  %d", len(elements))
	}
	e := elements[0]
	if e.Id != "namespace:kube-system/pod:heapster-ww9r6" {
		t.Errorf("element id is wrong")
	}
	if len(e.Tags()) != 9 {
		t.Errorf("element has wrong numbe of tags")
	}
	if len(e.Metrics()) != 2 {
		t.Errorf("element has wrong number of metrics")
	}
	if len(e.Samples()) != 2 {
		t.Errorf("element has wrong number of samples")
	}

}

func createDataBatch() *core.DataBatch {
	timestamp, _ := time.Parse(time.RFC3339, "2018-01-01T00:00:00+00:00")
	batch := core.DataBatch{
		Timestamp:  timestamp,
		MetricSets: make(map[string]*core.MetricSet),
	}

	batch.MetricSets["namespace:kube-system/pod:heapster-ww9r6"] = &core.MetricSet{
		Labels: map[string]string{
			"host_id":        "minikube",
			"hostname":       "minikube",
			"labels":         "addonmanager.kubernetes.io/mode:Reconcile,k8s-app:heapster",
			"namespace_id":   "efc44650-ffbd-11e7-a0bf-080027b171ba",
			"namespace_name": "kube-system",
			"nodename":       "minikube",
			"pod_id":         "761fc05e-008c-11e8-911d-080027b171ba",
			"pod_name":       "heapster-ww9r6",
			"type":           "pod",
		},
		MetricValues: map[string]core.MetricValue{
			"cpu/usage": {
				MetricType: core.MetricGauge,
				ValueType:  core.ValueInt64,
				IntValue:   9248344,
			},
		},
		LabeledMetrics: []core.LabeledMetric{
			{
				Name: "filesystem/usage",
				MetricValue: core.MetricValue{
					MetricType: core.MetricGauge,
					ValueType:  core.ValueInt64,
					IntValue:   835584,
				},
				Labels: map[string]string{
					"resource_id": "/dev/sda1",
				},
			},
		},
	}

	return &batch
}

func TestLinkElements(t *testing.T) {
	//given
	elements := createElements()
	//when
	LinkElements(elements)
	//then
	if len(elements) != 6 {
		t.Errorf("there should be 6 elements in elements, but actual =  %d", len(elements))
	}
	cluster := elements[0]
	podContainer := elements[1]
	pod := elements[2]
	node := elements[3]
	ns := elements[4]
	sysContainer := elements[5]
	if len(cluster.Relations()) != 0 {
		t.Errorf("cluster element should have no relations")
	}
	if len(podContainer.Relations()) != 0 {
		t.Errorf("pod_container element should have no relations")
	}
	if len(pod.Relations()) != 1 {
		t.Errorf("pod element should have 1 relation, but only has %d", len(pod.Relations()))
	}
	if len(node.Relations()) != 3 {
		t.Errorf("node elment should have 3 relations, but only has %d", len(node.Relations()))
	}
	if len(ns.Relations()) != 2 {
		t.Errorf("ns element should have 2 relations, but only has %d", len(ns.Relations()))
	}
	if len(sysContainer.Relations()) != 0 {
		t.Errorf("sys_container element should have no relations")
	}
}

func createElements() []metricly_core.Element {
	//cluster
	cluster := metricly_core.NewElement("cluster", "cluster", "cluster", "")
	cluster.AddTag("type", "cluster")

	//pod_container
	podContainer := metricly_core.NewElement("namespace:kube-system/pod:heapster-ww9r6/container:heapster", "namespace:kube-system/pod:heapster-ww9r6/container:heapster", "pod_container", "")
	podContainer.AddTag("type", "pod_container")
	podContainer.AddTag("pod_id", "761fc05e-008c-11e8-911d-080027b171ba")
	podContainer.AddTag("namespace_id", "efc44650-ffbd-11e7-a0bf-080027b171ba")
	podContainer.AddTag("host_id", "minikube")

	//pod
	pod := metricly_core.NewElement("namespace:kube-system/pod:heapster-ww9r6", "namespace:kube-system/pod:heapster-ww9r", "pod", "")
	pod.AddTag("type", "pod")
	pod.AddTag("pod_id", "761fc05e-008c-11e8-911d-080027b171ba")
	pod.AddTag("host_id", "minikube")
	pod.AddTag("namespace_id", "efc44650-ffbd-11e7-a0bf-080027b171ba")

	//node
	node := metricly_core.NewElement("node:minikube", "node:minikube", "node", "")
	node.AddTag("type", "node")
	node.AddTag("host_id", "minikube")

	//namespace
	ns := metricly_core.NewElement("namespace:kube-system", "namespace:kube-system", "ns", "")
	ns.AddTag("type", "ns")
	ns.AddTag("namespace_id", "efc44650-ffbd-11e7-a0bf-080027b171ba")

	//sys_container
	sysContainer := metricly_core.NewElement("node:minikube/container:init.scope", "node:minikube/container:init.scope", "sys_container", "")
	sysContainer.AddTag("type", "sys_container")
	sysContainer.AddTag("host_id", "minikube")

	return []metricly_core.Element{cluster, podContainer, pod, node, ns, sysContainer}
}
