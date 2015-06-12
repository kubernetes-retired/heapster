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

package repr

import (
	"encoding/json"
	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"net/http"
)

type ReprApi struct {
	cluster Cluster
}

// Create a new Api to serve from the specified cluster.
func NewReprApi(c Cluster) *ReprApi {
	return &ReprApi{
		cluster: c,
	}
}

// Register the Api on the appropriate endpoints.
func (a *ReprApi) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path("/api/repr/cluster").
		Doc("Exports all metrics at the cluster level").
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("").
		Filter(compressionFilter).
		To(a.exportAllClusterMetrics).
		Doc("Exports all metrics at the cluster level").
		Operation("allCluster").
		Writes(ClusterInfo{}))

	container.Add(ws)
}

func (a *ReprApi) exportAllClusterMetrics(request *restful.Request, response *restful.Response) {
	clinfo, _, err := a.cluster.GetAllClusterData()

	glog.Infof("Exported All Cluster Metrics")

	response.WriteAsJson(clinfo)
}

func compressionFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	// wrap responseWriter into a compressing one
	compress, err := restful.NewCompressingResponseWriter(resp.ResponseWriter, restful.ENCODING_GZIP)
	if err != nil {
		glog.Warningf("Failed to create CompressingResponseWriter for request %q: %v", req.Request.URL, err)
		return
	}
	resp.ResponseWriter = compress
	defer compress.Close()
	chain.ProcessFilter(req, resp)
}
