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

package sources

import (
	"k8s.io/heapster/sources/api"
)

const nodeContainerName = "machine"

var sysContainerNames = map[string]string{
	"/docker-daemon": "docker-daemon",
	"/kubelet":       "kubelet",
	"/kube-proxy":    "kube-proxy",
	"/system":        "system",
}

func isNode(c *api.Container) bool {
	return c.Name == "/"
}

func isSysContainer(c *api.Container) bool {
	_, exist := sysContainerNames[c.Name]
	return exist
}

func getSysContainerName(c *api.Container) string {
	return sysContainerNames[c.Name]
}
