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

package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/heapster/sources"
	"github.com/GoogleCloudPlatform/heapster/sources/api"
)

var (
	argMaster         = flag.String("kubernetes_master", "", "Kubernetes master IP")
	argMasterInsecure = flag.Bool("kubernetes_insecure", true, "Trust Kubernetes master certificate (if using https)")
	argKubeletPort    = flag.String("kubelet_port", "10250", "Kubelet port")
	// TODO: once known location for client auth is defined upstream, default this to that file.
	argClientAuthFile = flag.String("kubernetes_client_auth", "", "Kubernetes client authentication file")
	argKubeVersion    = flag.String("kubernetes_version", "v1beta1", "Kubernetes API version")
	argCadvisorPort   = flag.Int("cadvisor_port", 8080, "The port on which cadvisor binds to on all nodes.")
	argCoreOSMode     = flag.Bool("coreos", false, "When true, heapster looks will connect with fleet servers to get the list of nodes to monitor. It is expected that cadvisor will be running on all the nodes at the port specified using flag '--cadvisor_port'. Use flag '--fleet_endpoints' to manage fleet endpoints to watch.")
)

func newSources() ([]api.Source, error) {
	if len(*argMaster) > 0 {
		return sources.CreateKubeSources(*argMaster, *argKubeVersion, *argClientAuthFile, *argKubeletPort, *argMasterInsecure)
	}

	return sources.NewOtherSources(*argCadvisorPort, *argCoreOSMode)
}
