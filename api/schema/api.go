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

package schema

import (
	"github.com/GoogleCloudPlatform/heapster/api/schema/info"
	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
)

type SchemaApi struct {
	cluster Cluster
}

// Create a new Api to serve from the specified cluster.
func NewSchemaApi(c Cluster) *SchemaApi {
	return &SchemaApi{
		cluster: c,
	}
}

// Register the Api on the appropriate endpoints.
func (a *SchemaApi) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path("/api/schema/cluster").
		Doc("Exports all metrics at the cluster level").
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("").
		Filter(compressionFilter).
		To(a.exportAllClusterMetrics).
		Doc("Exports all metrics at the cluster level").
		Operation("allCluster").
		Writes(info.ClusterInfo{}))

	container.Add(ws)
}

func (a *SchemaApi) exportAllClusterMetrics(request *restful.Request, response *restful.Response) {
	clinfo, _, _ := a.cluster.GetAllClusterData()

	glog.V(2).Infof("Exported All Cluster Metrics")

	response.WriteAsJson(clinfo)
}

func compressionFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	// Wrap responseWriter into a compressing one
	compress, err := restful.NewCompressingResponseWriter(resp.ResponseWriter, restful.ENCODING_GZIP)
	if err != nil {
		glog.Warningf("Failed to create CompressingResponseWriter for request %q: %v", req.Request.URL, err)
		return
	}
	resp.ResponseWriter = compress
	defer compress.Close()
	chain.ProcessFilter(req, resp)
}
