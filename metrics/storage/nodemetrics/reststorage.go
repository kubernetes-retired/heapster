/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodemetrics

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/heapster/metrics/apis/metrics"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
)

var m1 = metrics.NodeMetrics{
	ObjectMeta: api.ObjectMeta{Name: "node-1"},
	Timestamp: unversioned.Time{},
	Window:    unversioned.Duration{time.Minute},

	Usage: api.ResourceList{
		api.ResourceCPU: *resource.NewQuantity(5000, resource.DecimalSI),
		api.ResourceMemory: *resource.NewQuantity(4000, resource.DecimalSI),
		api.ResourceNvidiaGPU: *resource.NewQuantity(3000, resource.DecimalSI),
		api.ResourcePods: *resource.NewQuantity(2000, resource.DecimalSI),
	},
}

var m2 = metrics.NodeMetrics{
	ObjectMeta: api.ObjectMeta{Name: "node-2"},
	Timestamp: unversioned.Time{},
	Window:    unversioned.Duration{time.Minute},

	Usage: api.ResourceList{
		api.ResourceCPU: *resource.NewQuantity(500, resource.DecimalSI),
		api.ResourceMemory: *resource.NewQuantity(400, resource.DecimalSI),
		api.ResourceNvidiaGPU: *resource.NewQuantity(300, resource.DecimalSI),
		api.ResourcePods: *resource.NewQuantity(200, resource.DecimalSI),
	},
}

var metricsList = metrics.NodeMetricsList{Items:[]metrics.NodeMetrics{m1, m2}}

type ReadOnlyStorage struct {
}

func NewReadOnlyStorage(resource unversioned.GroupResource) *ReadOnlyStorage {
	return &ReadOnlyStorage{}
}

func (s *ReadOnlyStorage) New() runtime.Object {
	return &metrics.NodeMetrics{}
}

// Get finds a resource in the storage by name and returns it.
// Although it can return an arbitrary error value, IsNotFound(err) is true for the
// returned error value err when the specified resource is not found.
func (s *ReadOnlyStorage) Get(ctx api.Context, name string) (runtime.Object, error) {
	glog.V(0).Infof("Get metrics for node: %v", name)
	for _, m := range metricsList.Items {
		if m.Name == name {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("Node %s not found", name)
}

// NewList returns an empty object that can be used with the List call.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
func (s *ReadOnlyStorage) NewList() runtime.Object {
	return &metrics.NodeMetricsList{}
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (s *ReadOnlyStorage) List(ctx api.Context, options *api.ListOptions) (runtime.Object, error) {
	return &metricsList, nil
}