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

package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/heapster/store"
)

// TestAggregateNodeMetricsEmpty tests the empty flow of aggregateNodeMetrics.
// The normal flow is tested through TestUpdate.
func TestAggregateNodeMetricsEmpty(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		c       = make(chan error)
		assert  = assert.New(t)
	)

	// Invocation with empty cluster
	go cluster.aggregateNodeMetrics(c)
	assert.NoError(<-c)
	assert.Empty(cluster.Nodes)
	assert.Empty(cluster.Metrics)
}

// TestAggregateKubeMetricsError tests the error flows of aggregateKubeMetrics.
// The normal flow is tested through TestUpdate.
func TestAggregateKubeMetricsEmpty(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		c       = make(chan error)
		assert  = assert.New(t)
	)

	// Invocation with empty cluster
	go cluster.aggregateKubeMetrics(c)
	assert.NoError(<-c)
	assert.Empty(cluster.Namespaces)

	// Error propagation from aggregateNamespaceMetrics
	cluster.Namespaces["default"] = nil
	go cluster.aggregateKubeMetrics(c)
	assert.Error(<-c)
	assert.NotEmpty(cluster.Namespaces)
}

// TestAggregateNamespaceMetrics tests the error flows of aggregateNamespaceMetrics.
// The normal flow is tested through TestUpdate.
func TestAggregateNamespaceMetricsError(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		c       = make(chan error)
		assert  = assert.New(t)
	)

	// Invocation with nil namespace
	go cluster.aggregateNamespaceMetrics(nil, c)
	assert.Error(<-c)
	assert.Empty(cluster.Namespaces)

	// Invocation for a namespace with no pods
	ns := cluster.addNamespace("default")
	go cluster.aggregateNamespaceMetrics(ns, c)
	assert.NoError(<-c)
	assert.Len(ns.Pods, 0)

	// Error propagation from aggregatePodMetrics
	ns.Pods["pod1"] = nil
	go cluster.aggregateNamespaceMetrics(ns, c)
	assert.Error(<-c)
}

// TestAggregatePodMetricsError tests the error flows of aggregatePodMetrics.
// The normal flow is tested through TestUpdate.
func TestAggregatePodMetricsError(t *testing.T) {
	var (
		cluster = newRealCluster(newTimeStore, time.Minute)
		c       = make(chan error)
		assert  = assert.New(t)
	)
	ns := cluster.addNamespace("default")
	node := cluster.addNode("newnode")
	pod := cluster.addPod("pod1", "uid111", ns, node)

	// Invocation with nil pod
	go cluster.aggregatePodMetrics(nil, c)
	assert.Error(<-c)

	// Invocation with empty pod
	go cluster.aggregatePodMetrics(pod, c)
	assert.NoError(<-c)

	// Invocation with a normal pod
	addContainerToMap("new_container", pod.Containers)
	addContainerToMap("new_container2", pod.Containers)
	go cluster.aggregatePodMetrics(pod, c)
	assert.NoError(<-c)
}

// TestAggregateMetricsError tests the error flows of aggregateMetrics.
func TestAggregateMetricsError(t *testing.T) {
	var (
		cluster    = newRealCluster(newTimeStore, time.Minute)
		targetInfo = InfoType{
			Metrics: make(map[string]*store.TimeStore),
			Labels:  make(map[string]string),
		}
		srcInfo = InfoType{
			Metrics: make(map[string]*store.TimeStore),
			Labels:  make(map[string]string),
		}
		assert = assert.New(t)
	)

	// Invocation with nil first argument
	sources := []*InfoType{&srcInfo}
	assert.Error(cluster.aggregateMetrics(nil, sources))

	// Invocation with empty second argument
	sources = []*InfoType{}
	assert.Error(cluster.aggregateMetrics(&targetInfo, sources))

	// Invocation with a nil element in the second argument
	sources = []*InfoType{&srcInfo, nil}
	assert.Error(cluster.aggregateMetrics(&targetInfo, sources))

	// Invocation with the target being also part of sources
	sources = []*InfoType{&srcInfo, &targetInfo}
	assert.Error(cluster.aggregateMetrics(&targetInfo, sources))
}

// TestAggregateMetricsNormal tests the normal flows of aggregateMetrics.
func TestAggregateMetricsNormal(t *testing.T) {
	var (
		cluster    = newRealCluster(newTimeStore, time.Minute)
		targetInfo = InfoType{
			Metrics: make(map[string]*store.TimeStore),
			Labels:  make(map[string]string),
		}
		srcInfo1 = InfoType{
			Metrics: make(map[string]*store.TimeStore),
			Labels:  make(map[string]string),
		}
		srcInfo2 = InfoType{
			Metrics: make(map[string]*store.TimeStore),
			Labels:  make(map[string]string),
		}
		now    = time.Now().Round(time.Minute)
		assert = assert.New(t)
	)

	newTS := newTimeStore()
	newTS.Put(store.TimePoint{
		Timestamp: now,
		Value:     uint64(5000),
	})
	newTS.Put(store.TimePoint{
		Timestamp: now.Add(time.Hour),
		Value:     uint64(3000),
	})
	newTS2 := newTimeStore()
	newTS2.Put(store.TimePoint{
		Timestamp: now,
		Value:     uint64(2000),
	})
	newTS2.Put(store.TimePoint{
		Timestamp: now.Add(time.Hour),
		Value:     uint64(3500),
	})
	newTS2.Put(store.TimePoint{
		Timestamp: now.Add(2 * time.Hour),
		Value:     uint64(9000),
	})

	srcInfo1.Metrics[memUsage] = &newTS
	srcInfo2.Metrics[memUsage] = &newTS2
	sources := []*InfoType{&srcInfo1, &srcInfo2}

	// Normal Invocation
	cluster.aggregateMetrics(&targetInfo, sources)

	assert.NotNil(targetInfo.Metrics[memUsage])
	targetMemTS := *(targetInfo.Metrics[memUsage])
	res := targetMemTS.Get(time.Time{}, time.Time{})
	assert.Len(res, 3)
	assert.Equal(res[0].Value, uint64(9000))
	assert.Equal(res[1].Value, uint64(6500))
	assert.Equal(res[2].Value, uint64(7000))
}
