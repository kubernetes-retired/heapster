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

package v1

import (
	"net/http"
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/golang/glog"

	"github.com/GoogleCloudPlatform/heapster/store"
)

// RegisterModel registers the Model API endpoints.
// All endpoints that end with a {metric-name} also receive a start time query parameter.
// The start time should be specified as a string, formatted according to RFC 3339.
func (a *Api) RegisterModel(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path("/api/v1/model").
		Doc("Root endpoint of the stats model").
		Consumes("*/*").
		Produces(restful.MIME_JSON)

	// The / endpoint returns a list of all the metrics that are available in the model
	ws.Route(ws.GET("/").
		To(a.availableMetrics).
		Filter(compressionFilter).
		Doc("Show which metrics are available").
		Operation("clusterMetrics"))

	// The /cluster/{metric-name} endpoint exposes an aggregated metric for the Cluster entity of the model.
	ws.Route(ws.GET("/cluster/{metric-name}").
		To(a.clusterMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated cluster-level metric").
		Operation("clusterMetrics").
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metric").DataType("string")).
		Writes(MetricResult{}))

	// The /nodes/{node-name}/{metric-name} endpoint exposes a metric for a Node entity of the model.
	// The {node-name} parameter is the hostname of a specific node.
	ws.Route(ws.GET("/nodes/{node-name}/{metric-name}").
		To(a.nodeMetrics).
		Filter(compressionFilter).
		Doc("Export a node-level metric").
		Operation("nodeMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metric").DataType("string")).
		Writes(MetricResult{}))

	// The /namespaces/{namespace-name}/{metric-name} endpoint exposes an aggregated metrics
	// for a Namespace entity of the model.
	ws.Route(ws.GET("/namespaces/{namespace-name}/{metric-name}").
		To(a.namespaceMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated namespace-level metric").
		Operation("namespaceMetrics").
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Writes(MetricResult{}))

	// The /namespaces/{namespace-name}/pods/{pod-name}/{metric-name} endpoint exposes
	// an aggregated metric for a Pod entity of the model.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/{metric-name}").
		To(a.podMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated pod-level metric").
		Operation("podMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to use").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to use").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Writes(MetricResult{}))

	// The /namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/{metric-name} endpoint exposes
	// a metric for a Container entity of the model.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/{metric-name}").
		To(a.podContainerMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated metric for a pod container").
		Operation("podContainerMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to use").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to use").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Writes(MetricResult{}))

	// The /nodes/{node-name}/freecontainers/{container-name}/{metric-name} endpoint exposes
	// a metric for a free Container entity of the model.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/{metric-name}").
		To(a.freeContainerMetrics).
		Filter(compressionFilter).
		Doc("Export a container-level metric for a free container").
		Operation("freeContainerMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to use").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the container to use").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Writes(MetricResult{}))

	container.Add(ws)
}

// availableMetrics returns a list of available metric names, to be reported to the /metrics endpoint.
// These metric names can be used to extract metrics from the various model entities.
func (a *Api) availableMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	result := cluster.GetAvailableMetrics()
	response.WriteEntity(result)
}

// clusterMetrics returns a metric timeseries for a metric of the Cluster entity.
func (a *Api) clusterMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	// Get metric name
	metric_name := request.PathParameter("metric-name")

	// Get start time, parse as time.Time
	req_stamp := parseRequestStartParam(request, response)

	timeseries, new_stamp, err := cluster.GetClusterMetric(metric_name, req_stamp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get cluster metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// nodeMetrics returns a metric timeseries for a metric of the Node entity.
func (a *Api) nodeMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()

	// Get node name (hostname)
	hostname := request.PathParameter("node-name")

	// Get metric name
	metric_name := request.PathParameter("metric-name")

	// Get start time, parse as time.Time
	req_stamp := parseRequestStartParam(request, response)

	timeseries, new_stamp, err := cluster.GetNodeMetric(hostname, metric_name, req_stamp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get cluster metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// namespaceMetrics returns a metric timeseries for a metric of the Namespace entity.
func (a *Api) namespaceMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()

	// Get namespace name
	namespace := request.PathParameter("namespace-name")

	// Get metric name
	metric_name := request.PathParameter("metric-name")

	// Get start time, parse as time.Time
	req_stamp := parseRequestStartParam(request, response)

	timeseries, new_stamp, err := cluster.GetNamespaceMetric(namespace, metric_name, req_stamp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get cluster metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// podMetrics returns a metric timeseries for a metric of the Pod entity.
func (a *Api) podMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()

	// Get namespace name
	namespace := request.PathParameter("namespace-name")

	// Get pod name
	pod := request.PathParameter("pod-name")

	// Get metric name
	metric_name := request.PathParameter("metric-name")

	// Get start time, parse as time.Time
	req_stamp := parseRequestStartParam(request, response)

	timeseries, new_stamp, err := cluster.GetPodMetric(namespace, pod, metric_name, req_stamp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get cluster metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// podContainerMetrics returns a metric timeseries for a metric of the Container entity.
// freeContainerMetrics addresses only pod containers, by using the namespace-name/pod-name/container-name path.
func (a *Api) podContainerMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()

	// Get namespace name
	namespace := request.PathParameter("namespace-name")

	// Get pod name
	pod := request.PathParameter("pod-name")

	// Get container name
	container := request.PathParameter("container-name")

	// Get metric name
	metric_name := request.PathParameter("metric-name")

	// Get start time, parse as time.Time
	req_stamp := parseRequestStartParam(request, response)

	timeseries, new_stamp, err := cluster.GetPodContainerMetric(namespace, pod, container, metric_name, req_stamp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get cluster metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// freeContainerMetrics returns a metric timeseries for a metric of the Container entity.
// freeContainerMetrics addresses only free containers, by using the node-name/container-name path.
func (a *Api) freeContainerMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()

	// Get node name
	node := request.PathParameter("node-name")

	// Get container name
	container := request.PathParameter("container-name")

	// Get metric name
	metric_name := request.PathParameter("metric-name")

	// Get start time, parse as time.Time
	req_stamp := parseRequestStartParam(request, response)

	timeseries, new_stamp, err := cluster.GetFreeContainerMetric(node, container, metric_name, req_stamp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get cluster metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// parseRequestStartParam obtains the time.Time to be used as the start time during a model Get method.
// parseRequestStartParam receives a request and a response as inputs, and returns the parsed time.
func parseRequestStartParam(request *restful.Request, response *restful.Response) time.Time {
	var err error
	query_param := request.QueryParameter("start")
	req_stamp := time.Time{}
	if query_param != "" {
		req_stamp, err = time.Parse(time.RFC3339, query_param)
		if err != nil {
			// Timestamp parameter cannot be parsed
			response.WriteError(http.StatusInternalServerError, err)
			glog.Errorf("timestamp argument cannot be parsed: %s", err)
			return time.Time{}
		}
	}
	return req_stamp
}

// exportTimeseries renders a []store.TimePoint and a timestamp into a MetricResult.
func exportTimeseries(ts []store.TimePoint, stamp time.Time) MetricResult {
	// Convert each store.TimePoint to a MetricPoint
	res_metrics := []MetricPoint{}
	for _, metric := range ts {
		newMP := MetricPoint{
			Timestamp: metric.Timestamp,
			Value:     metric.Value.(uint64),
		}
		res_metrics = append(res_metrics, newMP)
	}

	result := MetricResult{
		Metrics:         res_metrics,
		LatestTimestamp: stamp,
	}
	return result
}
