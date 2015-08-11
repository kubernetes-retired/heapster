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
	"errors"
	"net/http"
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/golang/glog"

	"k8s.io/heapster/model"
	"k8s.io/heapster/store"
)

// errModelNotActivated is the error that is returned when manager.cluster
// has not beed initialized.
var errModelNotActivated = errors.New("the model is not activated")

// RegisterModel registers the Model API endpoints.
// All endpoints that end with a {metric-name} also receive a start time query parameter.
// The start and end times should be specified as a string, formatted according to RFC 3339.
func (a *Api) RegisterModel(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path("/api/v1/model").
		Doc("Root endpoint of the stats model").
		Consumes("*/*").
		Produces(restful.MIME_JSON)

	// The / endpoint returns a list of all the entities that are available in the cluster
	ws.Route(ws.GET("/").
		To(a.allEntities).
		Filter(compressionFilter).
		Doc("Get a list of all entities available in the model").
		Operation("allEntities"))

	// The /metrics/ endpoint returns a list of all available metrics for the Cluster entity of the model.
	ws.Route(ws.GET("/metrics/").
		To(a.availableMetrics).
		Filter(compressionFilter).
		Doc("Get a list of all available metrics for the Cluster entity").
		Operation("availableMetrics"))

	// The /metrics/{metric-name} endpoint exposes an aggregated metric for the Cluster entity of the model.
	ws.Route(ws.GET("/metrics/{metric-name}").
		To(a.clusterMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated cluster-level metric").
		Operation("clusterMetrics").
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metric").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(MetricResult{}))

	// The /nodes/ endpoint returns a list of all Node entities in the cluster.
	ws.Route(ws.GET("/nodes/").
		To(a.allNodes).
		Filter(compressionFilter).
		Doc("Get a list of all Nodes in the model").
		Operation("allNodes").
		Writes(MetricResult{}))

	// The /nodes/{node-name} endpoint returns a list of all available API paths for a Node entity.
	ws.Route(ws.GET("/nodes/{node-name}").
		To(a.nodePaths).
		Filter(compressionFilter).
		Doc("Get a list of all available API paths for a Node entity").
		Operation("nodePaths").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")))

	// The /nodes/{node-name}/metrics endpoint returns a list of all available metrics for a Node entity.
	ws.Route(ws.GET("/nodes/{node-name}/metrics/").
		To(a.availableMetrics).
		Filter(compressionFilter).
		Doc("Get a list of all available metrics for a Node entity").
		Operation("availableMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")))

	// The /nodes/{node-name}/metrics/{metric-name} endpoint exposes a metric for a Node entity of the model.
	// The {node-name} parameter is the hostname of a specific node.
	ws.Route(ws.GET("/nodes/{node-name}/metrics/{metric-name}").
		To(a.nodeMetrics).
		Filter(compressionFilter).
		Doc("Export a node-level metric").
		Operation("nodeMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metric").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(MetricResult{}))

	// The /namespaces/ endpoint returns a list of all Namespace entities in the cluster.
	ws.Route(ws.GET("/namespaces/").
		To(a.allNamespaces).
		Filter(compressionFilter).
		Doc("Get a list of all Namespaces in the model").
		Operation("allNamespaces"))

	// The /namespaces/{namespace-name} endpoint returns a list of all available API Paths for a Namespace entity.
	ws.Route(ws.GET("/namespaces/{namespace-name}").
		To(a.namespacePaths).
		Filter(compressionFilter).
		Doc("Get a list of all available API paths for a namespace entity").
		Operation("namespacePaths").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

	// The /namespaces/{namespace-name}/metrics endpoint returns a list of all available metrics for a Namespace entity.
	ws.Route(ws.GET("/namespaces/{namespace-name}/metrics").
		To(a.availableMetrics).
		Filter(compressionFilter).
		Doc("Get a list of all available metrics for a Namespace entity").
		Operation("availableMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

	// The /namespaces/{namespace-name}/metrics/{metric-name} endpoint exposes an aggregated metrics
	// for a Namespace entity of the model.
	ws.Route(ws.GET("/namespaces/{namespace-name}/metrics/{metric-name}").
		To(a.namespaceMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated namespace-level metric").
		Operation("namespaceMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(MetricResult{}))

	// The /namespaces/{namespace-name}/pods endpoint returns a list of all Pod entities in the cluster,
	// under a specified namespace.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods").
		To(a.allPods).
		Filter(compressionFilter).
		Doc("Get a list of all Pods in the model, belonging to the specified Namespace").
		Operation("allPods").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

	// The /namespaces/{namespace-name}/pods/{pod-name} endpoint returns a list of all
	// API paths available for a pod
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}").
		To(a.podPaths).
		Filter(compressionFilter).
		Doc("Get a list of all API paths available for a Pod entity").
		Operation("podPaths").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

	// The /namespaces/{namespace-name}/pods/{pod-name}/metrics endpoint returns a list of all available metrics for a Pod entity.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/metrics").
		To(a.availableMetrics).
		Filter(compressionFilter).
		Doc("Get a list of all available metrics for a Pod entity").
		Operation("availableMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

	// The /namespaces/{namespace-name}/pods/{pod-name}/metrics/{metric-name} endpoint exposes
	// an aggregated metric for a Pod entity of the model.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/metrics/{metric-name}").
		To(a.podMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated pod-level metric").
		Operation("podMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(MetricResult{}))
	// The /namespaces/{namespace-name}/pods/{pod-name}/containers endpoint returns a list of all Container entities,
	// under a specified namespace and pod.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers").
		To(a.allPodContainers).
		Filter(compressionFilter).
		Doc("Get a list of all Containers in the model, belonging to the specified Namespace and Pod").
		Operation("allPodContainers").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

	// The /namespaces/{namespace-name}/pods/{pod-name}/containers/metrics/{container-name} endpoint
	// returns a list of all API paths available for a Pod Container
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}").
		To(a.containerPaths).
		Filter(compressionFilter).
		Doc("Get a list of all API paths available for a Pod Container entity").
		Operation("containerPaths").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

	// The /namespaces/{namespace-name}/pods/{pod-name}/containers/metrics/{container-name}/metrics endpoint
	// returns a list of all available metrics for a Pod Container entity.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/metrics").
		To(a.availableMetrics).
		Filter(compressionFilter).
		Doc("Get a list of all available metrics for a Pod entity").
		Operation("availableMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

	// The /namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/metrics/{metric-name} endpoint exposes
	// a metric for a Container entity of the model.
	ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/metrics/{metric-name}").
		To(a.podContainerMetrics).
		Filter(compressionFilter).
		Doc("Export an aggregated metric for a Pod Container").
		Operation("podContainerMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to use").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to use").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(MetricResult{}))

	// The /nodes/{node-name}/freecontainers/ endpoint returns a list of all free Container entities,
	// under a specified node.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/").
		To(a.allFreeContainers).
		Filter(compressionFilter).
		Doc("Get a list of all free Containers in the model, belonging to the specified Node").
		Operation("allFreeContainers").
		Param(ws.PathParameter("node-name", "The name of the namespace to lookup").DataType("string")))

	// The /nodes/{node-name}/freecontainers/{container-name}/ endpoint exposes
	// the available subpaths for a free container
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/").
		To(a.containerPaths).
		Filter(compressionFilter).
		Doc("Get a list of API paths for a free Container entity").
		Operation("freeContainerMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to use").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the container to use").DataType("string")).
		Writes(MetricResult{}))

	// The /nodes/{node-name}/freecontainers/{container-name}/metrics endpoint
	// returns a list of all available metrics for a Free Container entity.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/metrics").
		To(a.availableMetrics).
		Filter(compressionFilter).
		Doc("Get a list of all available metrics for a free Container entity").
		Operation("availableMetrics").
		Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

	// The /nodes/{node-name}/freecontainers/{container-name}/metrics/{metric-name} endpoint exposes
	// a metric for a free Container entity of the model.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/metrics/{metric-name}").
		To(a.freeContainerMetrics).
		Filter(compressionFilter).
		Doc("Export a container-level metric for a free container").
		Operation("freeContainerMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to use").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the container to use").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(MetricResult{}))

	container.Add(ws)
}

// allEntities returns a list of all the top-level paths that are available in the API.
func (a *Api) allEntities(request *restful.Request, response *restful.Response) {
	entities := []string{
		"metrics/",
		"namespaces/",
		"nodes/",
	}
	response.WriteEntity(entities)
}

// namespacePaths returns a list of all the available API paths that are available for a namespace.
func (a *Api) namespacePaths(request *restful.Request, response *restful.Response) {
	entities := []string{
		"pods/",
		"metrics/",
	}
	response.WriteEntity(entities)
}

// nodePaths returns a list of all the available API paths that are available for a node.
func (a *Api) nodePaths(request *restful.Request, response *restful.Response) {
	entities := []string{
		"freecontainers/",
		"metrics/",
	}
	response.WriteEntity(entities)
}

// podPaths returns a list of all the available API paths that are available for a pod.
func (a *Api) podPaths(request *restful.Request, response *restful.Response) {
	entities := []string{
		"containers/",
		"metrics/",
	}
	response.WriteEntity(entities)
}

// containerPaths returns a list of all the available API paths that are available for a container.
func (a *Api) containerPaths(request *restful.Request, response *restful.Response) {
	entities := []string{
		"metrics/",
	}
	response.WriteEntity(entities)
}

// allNodes returns a list of all the available node names in the cluster.
func (a *Api) allNodes(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}
	response.WriteEntity(cluster.GetNodes())
}

// allNamespaces returns a list of all the available namespaces in the cluster.
func (a *Api) allNamespaces(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}
	response.WriteEntity(cluster.GetNamespaces())
}

// allPods returns a list of all the available pods in the cluster.
func (a *Api) allPods(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}
	namespace := request.PathParameter("namespace-name")
	response.WriteEntity(cluster.GetPods(namespace))
}

// allPodContainers returns a list of all the available pod containers in the cluster.
func (a *Api) allPodContainers(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}
	namespace := request.PathParameter("namespace-name")
	pod := request.PathParameter("pod-name")
	response.WriteEntity(cluster.GetPodContainers(namespace, pod))
}

// allFreeContainers returns a list of all the available free containers in the cluster.
func (a *Api) allFreeContainers(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}
	node := request.PathParameter("node-name")
	response.WriteEntity(cluster.GetFreeContainers(node))
}

// availableMetrics returns a list of available metric names.
// These metric names can be used to extract metrics from the various model entities.
func (a *Api) availableMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}
	result := cluster.GetAvailableMetrics()
	response.WriteEntity(result)
}

// clusterMetrics returns a metric timeseries for a metric of the Cluster entity.
func (a *Api) clusterMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}

	timeseries, new_stamp, err := cluster.GetClusterMetric(model.ClusterRequest{
		MetricName: request.PathParameter("metric-name"),
		Start:      parseRequestParam("start", request, response),
		End:        parseRequestParam("end", request, response),
	})
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
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}

	timeseries, new_stamp, err := cluster.GetNodeMetric(model.NodeRequest{
		NodeName:   request.PathParameter("node-name"),
		MetricName: request.PathParameter("metric-name"),
		Start:      parseRequestParam("start", request, response),
		End:        parseRequestParam("end", request, response),
	})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get node metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// namespaceMetrics returns a metric timeseries for a metric of the Namespace entity.
func (a *Api) namespaceMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}

	timeseries, new_stamp, err := cluster.GetNamespaceMetric(model.NamespaceRequest{
		NamespaceName: request.PathParameter("namespace-name"),
		MetricName:    request.PathParameter("metric-name"),
		Start:         parseRequestParam("start", request, response),
		End:           parseRequestParam("end", request, response),
	})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get namespace metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// podMetrics returns a metric timeseries for a metric of the Pod entity.
func (a *Api) podMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}

	timeseries, new_stamp, err := cluster.GetPodMetric(model.PodRequest{
		NamespaceName: request.PathParameter("namespace-name"),
		PodName:       request.PathParameter("pod-name"),
		MetricName:    request.PathParameter("metric-name"),
		Start:         parseRequestParam("start", request, response),
		End:           parseRequestParam("end", request, response),
	})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get pod metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// podContainerMetrics returns a metric timeseries for a metric of the Container entity.
// freeContainerMetrics addresses only pod containers, by using the namespace-name/pod-name/container-name path.
func (a *Api) podContainerMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}

	timeseries, new_stamp, err := cluster.GetPodContainerMetric(model.PodContainerRequest{
		NamespaceName: request.PathParameter("namespace-name"),
		PodName:       request.PathParameter("pod-name"),
		ContainerName: request.PathParameter("container-name"),
		MetricName:    request.PathParameter("metric-name"),
		Start:         parseRequestParam("start", request, response),
		End:           parseRequestParam("end", request, response),
	})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get pod container metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// freeContainerMetrics returns a metric timeseries for a metric of the Container entity.
// freeContainerMetrics addresses only free containers, by using the node-name/container-name path.
func (a *Api) freeContainerMetrics(request *restful.Request, response *restful.Response) {
	cluster := a.manager.GetCluster()
	if cluster == nil {
		response.WriteError(400, errModelNotActivated)
		return
	}

	timeseries, new_stamp, err := cluster.GetFreeContainerMetric(model.FreeContainerRequest{
		NodeName:      request.PathParameter("node-name"),
		ContainerName: request.PathParameter("container-name"),
		MetricName:    request.PathParameter("metric-name"),
		Start:         parseRequestParam("start", request, response),
		End:           parseRequestParam("end", request, response),
	})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		glog.Errorf("unable to get free container metric: %s", err)
		return
	}
	response.WriteEntity(exportTimeseries(timeseries, new_stamp))
}

// parseRequestParam parses a time.Time from a named QueryParam.
// parseRequestParam receives a request and a response as inputs, and returns the parsed time.
func parseRequestParam(param string, request *restful.Request, response *restful.Response) time.Time {
	var err error
	query_param := request.QueryParameter(param)
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
