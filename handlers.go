// Copyright 2014 Google Inc. All Rights Reserved.
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
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/GoogleCloudPlatform/heapster/api/repr"
	"github.com/GoogleCloudPlatform/heapster/api/v1"
	"github.com/GoogleCloudPlatform/heapster/manager"
	"github.com/GoogleCloudPlatform/heapster/sinks"
	"github.com/GoogleCloudPlatform/heapster/sources/api"
	"github.com/GoogleCloudPlatform/heapster/validate"
	restful "github.com/emicklei/go-restful"
)

const pprofBasePath = "/debug/pprof/"

func setupHandlers(sources []api.Source, sink sinks.ExternalSinkManager, m manager.Manager) http.Handler {
	// Make API handler.
	wsContainer := restful.NewContainer()
	a := v1.NewApi(m)
	a.Register(wsContainer)

	// Make ReprApi handler.
	reprApi := repr.NewReprApi(m.GetCluster())
	reprApi.Register(wsContainer)

	// Validation/Debug handler.
	handleValidate := func(req *restful.Request, resp *restful.Response) {
		err := validate.HandleRequest(resp, sources, sink)
		if err != nil {
			fmt.Fprintf(resp, "%s", err)
		}
	}
	ws := new(restful.WebService).
		Path("/validate").
		Produces("text/plain")
	ws.Route(ws.GET("").To(handleValidate)).
		Doc("get validation information")
	wsContainer.Add(ws)

	// TODO(jnagal): Add a main status page.
	// Redirect root to /validate
	redirectHandler := http.RedirectHandler(validate.ValidatePage, http.StatusTemporaryRedirect)
	handleRoot := func(req *restful.Request, resp *restful.Response) {
		redirectHandler.ServeHTTP(resp, req.Request)
	}
	ws = new(restful.WebService)
	ws.Route(ws.GET("/").To(handleRoot))
	wsContainer.Add(ws)

	handlePprofEndpoint := func(req *restful.Request, resp *restful.Response) {
		name := strings.TrimPrefix(req.Request.URL.Path, pprofBasePath)
		switch name {
		case "profile":
			pprof.Profile(resp, req.Request)
		case "symbol":
			pprof.Symbol(resp, req.Request)
		case "cmdline":
			pprof.Cmdline(resp, req.Request)
		default:
			pprof.Index(resp, req.Request)
		}
	}

	// Setup pporf handlers.
	ws = new(restful.WebService).Path(pprofBasePath)
	ws.Route(ws.GET("/{subpath:*}").To(func(req *restful.Request, resp *restful.Response) {
		handlePprofEndpoint(req, resp)
	})).Doc("pprof endpoint")
	wsContainer.Add(ws)

	return wsContainer
}
