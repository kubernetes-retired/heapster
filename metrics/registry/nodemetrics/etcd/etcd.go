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

package etcd

import (
	"fmt"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/registry/generic/registry"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/validation/field"
	"k8s.io/heapster/metrics/apis/metrics"
)

type REST struct {
	*registry.Store
}

// NewREST returns a RESTStorage object that will work against node metrics.
func NewREST(opts generic.RESTOptions) *REST {
	prefix := "/nodemetrics"

	newListFunc := func() runtime.Object { return &metrics.NodeMetricsList{} }
	storageInterface, _ := opts.Decorator(
		opts.StorageConfig,
		100,
		&metrics.NodeMetrics{},
		prefix,
		Strategy,
		newListFunc,
		storage.NoTriggerPublisher,
	)

	store := &registry.Store{
		NewFunc:     func() runtime.Object { return &metrics.NodeMetrics{} },
		NewListFunc: newListFunc,
		KeyRootFunc: func(ctx api.Context) string {
			return prefix
		},
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return registry.NoNamespaceKeyFunc(ctx, prefix, name)
		},
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*metrics.NodeMetrics).Name, nil
		},
		PredicateFunc:           MatchNodeMetrics,
		QualifiedResource:       metrics.Resource("nodemetrics"),
		DeleteCollectionWorkers: opts.DeleteCollectionWorkers,

		CreateStrategy: Strategy,
		UpdateStrategy: Strategy,
		DeleteStrategy: Strategy,

		ReturnDeletedObject: true,

		Storage: storageInterface,
	}

	return &REST{store}
}

var Strategy = nodeMetricsStrategy{api.Scheme, api.SimpleNameGenerator}

type nodeMetricsStrategy struct {
	runtime.ObjectTyper
	api.NameGenerator
}

func (nodeMetricsStrategy) NamespaceScoped() bool {
	return false
}

func MatchNodeMetrics(label labels.Selector, field fields.Selector) *generic.SelectionPredicate {
	return &generic.SelectionPredicate{
		Label: label,
		Field: field,
		GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
			cluster, ok := obj.(*metrics.NodeMetrics)
			if !ok {
				return nil, nil, fmt.Errorf("given object is not a node metric.")
			}
			return labels.Set(cluster.ObjectMeta.Labels), ClusterToSelectableFields(cluster), nil
		},
	}
}

func ClusterToSelectableFields(nodeMetrics *metrics.NodeMetrics) fields.Set {
	return ObjectMetaFieldsSet(nodeMetrics.ObjectMeta, false)
}

// ObjectMetaFieldsSet returns a fields set that represents the ObjectMeta.
func ObjectMetaFieldsSet(objectMeta api.ObjectMeta, hasNamespaceField bool) fields.Set {
	if !hasNamespaceField {
		return fields.Set{
			"metadata.name": objectMeta.Name,
		}
	}
	return fields.Set{
		"metadata.name":      objectMeta.Name,
		"metadata.namespace": objectMeta.Namespace,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (nodeMetricsStrategy) PrepareForCreate(ctx api.Context, obj runtime.Object) {
	//obj.(*metrics_api.NodeMetrics)
}

// Validate validates a new cluster.
func (nodeMetricsStrategy) Validate(ctx api.Context, obj runtime.Object) field.ErrorList {
	//cluster := obj.(*federation.Cluster)
	//return validation.ValidateCluster(cluster)
	return field.ErrorList{}
}

// Canonicalize normalizes the object after validation.
func (nodeMetricsStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for cluster.
func (nodeMetricsStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (nodeMetricsStrategy) PrepareForUpdate(ctx api.Context, obj, old runtime.Object) {
	//obj.(*metrics_api.NodeMetrics)
	//old.(*metrics_api.NodeMetrics)
}

// ValidateUpdate is the default update validation for an end user.
func (nodeMetricsStrategy) ValidateUpdate(ctx api.Context, obj, old runtime.Object) field.ErrorList {
	//return validation.ValidateClusterUpdate(obj.(*federation.Cluster), old.(*federation.Cluster))
	return field.ErrorList{}
}
func (nodeMetricsStrategy) AllowUnconditionalUpdate() bool {
	return true
}