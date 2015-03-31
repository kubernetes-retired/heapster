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
	labelPodId         = "pod_id"
	labelPodName       = "pod_name"
	labelContainerName = "container_name"
	labelLabels        = "labels"
	labelHostname      = "hostname"
	labelResourceID    = "resource_id"
)

// TODO(vmarmol): Things we should consider adding (note that we only get 10 labels):
// - POD name, container name, and host IP: Useful to users but maybe we should just mangle them with ID and IP
// - Namespace: Are IDs unique only per namespace? If so, mangle it into the ID.
var allLabels = []LabelDescriptor{
	{
		Key:         labelPodId,
		Description: "The unique ID of the pod",
	},
	{
		Key:         labelPodName,
		Description: "The name of the pod",
	},
	{
		Key:         labelContainerName,
		Description: "User-provided name of the container or full container name for system containers",
	},
	{
		Key:         labelLabels,
		Description: "Comma-separated list of user-provided labels",
	},
	{
		Key:         labelHostname,
		Description: "Hostname where the container ran",
	},
	{
		Key:         labelResourceID,
		Description: "Identifier(s) specific to a metric",
	},
}

func SupportedLabels() []LabelDescriptor {
	return allLabels
}

func SupportedLabelKeys() []string {
	keys := []string{}
	for _, label := range allLabels {
		keys = append(keys, label.Key)
	}
	return keys
}
