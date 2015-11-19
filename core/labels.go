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

package core

// Definition of labels supported in MetricSet.

var (
	LabelMetricSetType = LabelDescriptor{
		Key:         "type",
		Description: "Type of the metrcis set (container, pod, namespace, node, cluster)",
	}
	MetricSetTypeSystemContainer = "sys_container"
	MetricSetTypePodContainer    = "pod_container"
	MetricSetTypePod             = "pod"
	MetricSetTypeNamespace       = "ns"
	MetricSetTypeNode            = "node"
	MetricSetTypeCluster         = "cluster"

	LabelPodId = LabelDescriptor{
		Key:         "pod_id",
		Description: "The unique ID of the pod",
	}
	LabelPodName = LabelDescriptor{
		Key:         "pod_name",
		Description: "The name of the pod",
	}
	LabelPodNamespace = LabelDescriptor{
		Key:         "pod_namespace",
		Description: "The namespace of the pod",
	}
	LabelPodNamespaceUID = LabelDescriptor{
		Key:         "namespace_id",
		Description: "The UID of namespace of the pod",
	}
	LabelContainerName = LabelDescriptor{
		Key:         "container_name",
		Description: "User-provided name of the container or full container name for system containers",
	}
	LabelLabels = LabelDescriptor{
		Key:         "labels",
		Description: "Comma-separated list of user-provided labels",
	}
	LabelHostname = LabelDescriptor{
		Key:         "hostname",
		Description: "Hostname where the container ran",
	}
	LabelResourceID = LabelDescriptor{
		Key:         "resource_id",
		Description: "Identifier(s) specific to a metric",
	}
	LabelHostID = LabelDescriptor{
		Key:         "host_id",
		Description: "Identifier specific to a host. Set by cloud provider or user",
	}
	LabelContainerBaseImage = LabelDescriptor{
		Key:         "container_base_image",
		Description: "User-defined image name that is run inside the container",
	}
)

type LabelDescriptor struct {
	// Key to use for the label.
	Key string `json:"key,omitempty"`

	// Description of the label.
	Description string `json:"description,omitempty"`
}
