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

package api

const (
	LabelPodId           = "pod_id"
	LabelPodName         = "pod_name"
	LabelPodNamespace    = "pod_namespace"
	LabelPodNamespaceUID = "namespace_id"
	LabelContainerName   = "container_name"
	LabelLabels          = "labels"
	LabelHostname        = "hostname"
	LabelResourceID      = "resource_id"
	LabelExternalID      = "external_id"
)

// TODO(vmarmol): Things we should consider adding (note that we only get 10 labels):
// - POD name, container name, and host IP: Useful to users but maybe we should just mangle them with ID and IP
// - Namespace: Are IDs unique only per namespace? If so, mangle it into the ID.
var allLabels = []LabelDescriptor{
	{
		Key:         LabelPodId,
		Description: "The unique ID of the pod",
	},
	{
		Key:         LabelPodName,
		Description: "The name of the pod",
	},
	{
		Key:         LabelContainerName,
		Description: "User-provided name of the container or full container name for system containers",
	},
	{
		Key:         LabelLabels,
		Description: "Comma-separated list of user-provided labels",
	},
	{
		Key:         LabelHostname,
		Description: "Hostname where the container ran",
	},
	{
		Key:         LabelPodNamespace,
		Description: "The namespace of the pod",
	},
	{
		Key:         LabelPodNamespaceUID,
		Description: "The UID of namespace of the pod",
	},
	{
		Key:         LabelResourceID,
		Description: "Identifier(s) specific to a metric",
	},
	{
		Key:         LabelExternalID,
		Description: "Identifier specific to a node. Set by cloud provider or user",
	},
}

func SupportedLabels() []LabelDescriptor {
	return allLabels
}
