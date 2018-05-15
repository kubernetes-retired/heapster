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
	"regexp"
	"testing"
	"time"

	metricly_core "github.com/metricly/go-client/model/core"
	"k8s.io/heapster/common/metricly"
	"k8s.io/heapster/metrics/core"
)

func TestShortenNameWithoutType(t *testing.T) {
	//given
	fqn := "cluster"
	//when & then
	if shortenName(fqn) != "cluster" {
		t.Errorf("Shorten fqn failed on single name with no type")
	}
}

func TestPrettyElementType(t *testing.T) {
	if prettyElementType("cluster") != "Kubernetes Cluster" {
		t.Errorf("pretty element type for 'cluster' failed to match expected 'Kubernetes Cluster'")
	}
	if prettyElementType("ns") != "Kubernetes Namespace" {
		t.Errorf("pretty element type for 'ns' failed to match expected 'Kubernetes Namespace'")
	}
	if prettyElementType("node") != "Kubernetes Node" {
		t.Errorf("pretty element type for 'node' failed to match expected 'Kubernetes Node'")
	}
	if prettyElementType("pod") != "Kubernetes Pod" {
		t.Errorf("pretty element type for 'pod' failed to match expected 'Kubernetes Pod'")
	}
	if prettyElementType("pod_container") != "Kubernetes Pod Container" {
		t.Errorf("pretty element type for 'pod_container' failed to match expected 'Kubernetes Pod Container'")
	}
	if prettyElementType("sys_container") != "Kubernetes Sys Container" {
		t.Errorf("pretty element type for 'sys_container' failed to match expected 'Kubernetes Sys Container'")
	}
}
func TestConvertDataBatchToElements(t *testing.T) {
	//given
	batch := createDataBatch()
	//when
	elements := DataBatchToElements(metricly.MetriclyConfig{}, batch)
	//then
	if len(elements) != 1 {
		t.Errorf("There should be 1 element in elements, but actual =  %d", len(elements))
	}
	e := elements[0]
	if e.Id != "namespace:kube-system/pod:heapster-ww9r6" {
		t.Errorf("The element id is wrong")
	}
	if e.Name != "kube-system/heapster-ww9r6" {
		t.Errorf("The element name is not propertly shortened")
	}
	if len(e.Tags()) != 10 {
		t.Errorf("The element has the wrong number of tags")
	}
	tagMode, found := e.Tag("addonmanager.kubernetes.io/mode")
	if !found || tagMode.Value != "Reconcile" {
		t.Errorf("Custom Label 'addonmanager.kubernetes.io/mode' is not converted to an expected element tag")
	}
	tagApp, found := e.Tag("k8s-app")
	if !found || tagApp.Value != "heapster" {
		t.Errorf("Custom Label 'k8s-app' is not converted to an expected element tag")
	}
	if len(e.Metrics()) != 2 {
		t.Errorf("The element has the wrong number of metrics")
	}
	if len(e.Samples()) != 2 {
		t.Errorf("The element has the wrong number of samples")
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

func TestInclusionFilter(t *testing.T) {
	//given
	re, _ := regexp.Compile("pod.*")
	filters := []metricly.Filter{
		{
			Type:  "label",
			Name:  "type",
			Regex: re,
		},
	}
	ms := &core.MetricSet{
		Labels: map[string]string{
			"type": "pod",
		},
	}
	//when & then
	if !include(filters, ms) {
		t.Errorf("The metric set should be included and return true")
	}
}

func TestExclusionFilter(t *testing.T) {
	re, _ := regexp.Compile("sys_.*")
	filters := []metricly.Filter{
		{
			Type:  "label",
			Name:  "type",
			Regex: re,
		},
	}
	ms := &core.MetricSet{
		Labels: map[string]string{
			"type": "sys_container",
		},
	}
	if !exclude(filters, ms) {
		t.Errorf("The metric set should be excluded and return true")
	}
}

func TestFilter(t *testing.T) {
	//given
	ire, _ := regexp.Compile("pod.*")
	ifilters := []metricly.Filter{
		{
			Type:  "label",
			Name:  "type",
			Regex: ire,
		},
	}
	ere, _ := regexp.Compile("pod_container")
	efilters := []metricly.Filter{
		{
			Type:  "label",
			Name:  "type",
			Regex: ere,
		},
	}
	pod := &core.MetricSet{
		Labels: map[string]string{
			"type": "pod",
		},
	}
	container := &core.MetricSet{
		Labels: map[string]string{
			"type": "pod_container",
		},
	}
	//when & then
	if !filter(ifilters, efilters, pod) {
		t.Errorf("pod should be pass the filter")
	}
	if filter(ifilters, efilters, container) {
		t.Errorf("pod container should not pass the filter")
	}
}

func TestLinkElements(t *testing.T) {
	//given
	elements := createElements()
	//when
	LinkElements(elements)
	//then
	if len(elements) != 6 {
		t.Errorf("There should be 6 elements in elements, but actual =  %d", len(elements))
	}
	cluster := elements[0]
	podContainer := elements[1]
	pod := elements[2]
	node := elements[3]
	ns := elements[4]
	sysContainer := elements[5]
	if len(cluster.Relations()) != 0 {
		t.Errorf("The cluster element should have no relations")
	}
	if len(podContainer.Relations()) != 0 {
		t.Errorf("The pod_container element should have no relations")
	}
	if len(pod.Relations()) != 1 {
		t.Errorf("The pod element should have 1 relation, but only has %d", len(pod.Relations()))
	}
	if len(node.Relations()) != 3 {
		t.Errorf("The node elment should have 3 relations, but only has %d", len(node.Relations()))
	}
	if len(ns.Relations()) != 2 {
		t.Errorf("The ns element should have 2 relations, but only has %d", len(ns.Relations()))
	}
	if len(sysContainer.Relations()) != 0 {
		t.Errorf("The sys_container element should have no relations")
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
