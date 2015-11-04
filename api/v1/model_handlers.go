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

	restful "github.com/emicklei/go-restful"

	"k8s.io/heapster/api/v1/types"
)

// errModelNotActivated is the error that is returned by the API handlers
// when manager.model has not been initialized.
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

	// The / endpoint returns a list of all the entities that are available in the model
	ws.Route(ws.GET("/").
		To(a.allEntities).
		Doc("Get a list of all entities available in the model").
		Operation("allEntities"))

	// The /metrics/ endpoint returns a list of all available metrics for the Cluster entity of the model.
	ws.Route(ws.GET("/metrics/").
		To(a.availableMetrics).
		Doc("Get a list of all available metrics for the Cluster entity").
		Operation("availableMetrics"))

	// The /stats/ endpoint returns a list of all available stats for the Cluster entity of the model.
	ws.Route(ws.GET("/stats/").
		To(a.clusterStats).
		Doc("Get all available stats for the Cluster entity").
		Operation("clusterStats"))

	// The /metrics/{metric-name} endpoint exposes an aggregated metric for the Cluster entity of the model.
	ws.Route(ws.GET("/metrics/{metric-name}").
		To(a.clusterMetrics).
		Doc("Export an aggregated cluster-level metric").
		Operation("clusterMetrics").
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metric").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(types.MetricResult{}))

	// The /nodes/ endpoint returns a list of all Node entities in the cluster.
	ws.Route(ws.GET("/nodes/").
		To(a.allNodes).
		Doc("Get a list of all Nodes in the model").
		Operation("allNodes").
		Writes(types.MetricResult{}))

	// The /nodes/{node-name} endpoint returns a list of all available API paths for a Node entity.
	ws.Route(ws.GET("/nodes/{node-name}").
		To(a.nodePaths).
		Doc("Get a list of all available API paths for a Node entity").
		Operation("nodePaths").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")))

	// The /nodes/{node-name}/stats endpoint returns all available derived stats for a Node entity.
	ws.Route(ws.GET("/nodes/{node-name}/stats/").
		To(a.nodeStats).
		Doc("Get all available stats for a Node entity.").
		Operation("nodeStats").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")))

	// The /nodes/{node-name}/metrics endpoint returns a list of all available metrics for a Node entity.
	ws.Route(ws.GET("/nodes/{node-name}/metrics/").
		To(a.availableMetrics).
		Doc("Get a list of all available metrics for a Node entity").
		Operation("availableMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")))

	// The /nodes/{node-name}/metrics/{metric-name} endpoint exposes a metric for a Node entity of the model.
	// The {node-name} parameter is the hostname of a specific node.
	ws.Route(ws.GET("/nodes/{node-name}/metrics/{metric-name}").
		To(a.nodeMetrics).
		Doc("Export a node-level metric").
		Operation("nodeMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to lookup").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metric").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(types.MetricResult{}))

	if a.runningInKubernetes {
		// The /namespaces/ endpoint returns a list of all Namespace entities in the model.
		ws.Route(ws.GET("/namespaces/").
			To(a.allNamespaces).
			Doc("Get a list of all Namespaces in the model").
			Operation("allNamespaces"))

		// The /namespaces/{namespace-name} endpoint returns a list of all available API Paths for a Namespace entity.
		ws.Route(ws.GET("/namespaces/{namespace-name}").
			To(a.namespacePaths).
			Doc("Get a list of all available API paths for a namespace entity").
			Operation("namespacePaths").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/stats endpoint returns all available derived stats for a Namespace entity.
		ws.Route(ws.GET("/namespaces/{namespace-name}/stats/").
			To(a.namespaceStats).
			Doc("Get all available stats for a Namespace entity.").
			Operation("namespaceStats").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/metrics endpoint returns a list of all available metrics for a Namespace entity.
		ws.Route(ws.GET("/namespaces/{namespace-name}/metrics").
			To(a.availableMetrics).
			Doc("Get a list of all available metrics for a Namespace entity").
			Operation("availableMetrics").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/metrics/{metric-name} endpoint exposes an aggregated metrics
		// for a Namespace entity of the model.
		ws.Route(ws.GET("/namespaces/{namespace-name}/metrics/{metric-name}").
			To(a.namespaceMetrics).
			Doc("Export an aggregated namespace-level metric").
			Operation("namespaceMetrics").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
			Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
			Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
			Writes(types.MetricResult{}))

		// The /namespaces/{namespace-name}/pods endpoint returns a list of all Pod entities in the model,
		// under a specified namespace.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods").
			To(a.allPods).
			Doc("Get a list of all Pods in the model, belonging to the specified Namespace").
			Operation("allPods").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name} endpoint returns a list of all
		// API paths available for a pod
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}").
			To(a.podPaths).
			Doc("Get a list of all API paths available for a Pod entity").
			Operation("podPaths").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name}/stats endpoint returns all available derived stats for a Pod entity.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/stats/").
			To(a.podStats).
			Doc("Get all available stats for a Pod entity.").
			Operation("podStats").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name}/metrics endpoint returns a list of all available metrics for a Pod entity.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/metrics").
			To(a.availableMetrics).
			Doc("Get a list of all available metrics for a Pod entity").
			Operation("availableMetrics").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name}/metrics/{metric-name} endpoint exposes
		// an aggregated metric for a Pod entity of the model.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/metrics/{metric-name}").
			To(a.podMetrics).
			Doc("Export an aggregated pod-level metric").
			Operation("podMetrics").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
			Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
			Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
			Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
			Writes(types.MetricResult{}))
		// The /namespaces/{namespace-name}/pods/{pod-name}/containers endpoint returns a list of all Container entities,
		// under a specified namespace and pod.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers").
			To(a.allPodContainers).
			Doc("Get a list of all Containers in the model, belonging to the specified Namespace and Pod").
			Operation("allPodContainers").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name} endpoint
		// returns a list of all API paths available for a Pod Container
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}").
			To(a.containerPaths).
			Doc("Get a list of all API paths available for a Pod Container entity").
			Operation("containerPaths").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
			Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/stats endpoint returns derived stats for a Pod Container entity.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/stats/").
			To(a.podContainerStats).
			Doc("Get all available stats for a Pod Container entity.").
			Operation("podContainerStats").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
			Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name}/containers/metrics/{container-name}/metrics endpoint
		// returns a list of all available metrics for a Pod Container entity.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/metrics").
			To(a.availableMetrics).
			Doc("Get a list of all available metrics for a Pod entity").
			Operation("availableMetrics").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to lookup").DataType("string")).
			Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

		// The /namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/metrics/{metric-name} endpoint exposes
		// a metric for a Container entity of the model.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pods/{pod-name}/containers/{container-name}/metrics/{metric-name}").
			To(a.podContainerMetrics).
			Doc("Export an aggregated metric for a Pod Container").
			Operation("podContainerMetrics").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to use").DataType("string")).
			Param(ws.PathParameter("pod-name", "The name of the pod to use").DataType("string")).
			Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")).
			Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
			Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
			Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
			Writes(types.MetricResult{}))

		// The /nodes/{node-name}/pods/ endpoint returns a list of all Pods entities under a specified node.
		ws.Route(ws.GET("/nodes/{node-name}/pods/").
			To(a.nodePods).
			Doc("Get a list of all Pods belonging to a specified Node in the model").
			Operation("nodePods").
			Param(ws.PathParameter("node-name", "The name of the namespace to lookup").DataType("string")))
	}

	// The /nodes/{node-name}/freecontainers/ endpoint returns a list of all free Container entities,
	// under a specified node.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/").
		To(a.allFreeContainers).
		Doc("Get a list of all free Containers in the model, belonging to the specified Node").
		Operation("allFreeContainers").
		Param(ws.PathParameter("node-name", "The name of the namespace to lookup").DataType("string")))

	// The /nodes/{node-name}/freecontainers/{container-name}/ endpoint exposes
	// the available subpaths for a free container
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/").
		To(a.containerPaths).
		Doc("Get a list of API paths for a free Container entity").
		Operation("freeContainerMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to use").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the container to use").DataType("string")).
		Writes(types.MetricResult{}))

	// The /nodes/{node-name}/freecontainers/{container-name}/stats endpoint returns derived stats for a Free Container entity.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/stats").
		To(a.freeContainerStats).
		Doc("Get all available stats for a Free Container entity.").
		Operation("freeContainerStats").
		Param(ws.PathParameter("node-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

	// The /nodes/{node-name}/freecontainers/{container-name}/metrics endpoint
	// returns a list of all available metrics for a Free Container entity.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/metrics").
		To(a.availableMetrics).
		Doc("Get a list of all available metrics for a free Container entity").
		Operation("availableMetrics").
		Param(ws.PathParameter("node-name", "The name of the namespace to lookup").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the namespace to use").DataType("string")))

	// The /nodes/{node-name}/freecontainers/{container-name}/metrics/{metric-name} endpoint exposes
	// a metric for a free Container entity of the model.
	ws.Route(ws.GET("/nodes/{node-name}/freecontainers/{container-name}/metrics/{metric-name}").
		To(a.freeContainerMetrics).
		Doc("Export a container-level metric for a free container").
		Operation("freeContainerMetrics").
		Param(ws.PathParameter("node-name", "The name of the node to use").DataType("string")).
		Param(ws.PathParameter("container-name", "The name of the container to use").DataType("string")).
		Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
		Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
		Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
		Writes(types.MetricResult{}))

	if a.runningInKubernetes {
		// The /namespaces/{namespace-name}/pod-list/{pod-list}/metrics/{metric-name} endpoint exposes
		// metrics for a list od pods of the model.
		ws.Route(ws.GET("/namespaces/{namespace-name}/pod-list/{pod-list}/metrics/{metric-name}").
			To(a.podListMetrics).
			Doc("Export a metric for all pods from the given list").
			Operation("podListMetric").
			Param(ws.PathParameter("namespace-name", "The name of the namespace to lookup").DataType("string")).
			Param(ws.PathParameter("pod-list", "Comma separated list of pod names to lookup").DataType("string")).
			Param(ws.PathParameter("metric-name", "The name of the requested metric").DataType("string")).
			Param(ws.QueryParameter("start", "Start time for requested metrics").DataType("string")).
			Param(ws.QueryParameter("end", "End time for requested metric").DataType("string")).
			Writes(types.MetricResult{}))
	}

	container.Add(ws)
}

// allEntities returns a list of all the top-level paths that are available in the API.
func (a *Api) allEntities(request *restful.Request, response *restful.Response) {
	entities := []string{
		"metrics/",
		"stats/",
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
		"stats/",
	}
	response.WriteEntity(entities)
}

// nodePaths returns a list of all the available API paths that are available for a node.
func (a *Api) nodePaths(request *restful.Request, response *restful.Response) {
	entities := []string{
		"freecontainers/",
		"pods/",
		"metrics/",
		"stats/",
	}
	response.WriteEntity(entities)
}

// podPaths returns a list of all the available API paths that are available for a pod.
func (a *Api) podPaths(request *restful.Request, response *restful.Response) {
	entities := []string{
		"containers/",
		"metrics/",
		"stats/",
	}
	response.WriteEntity(entities)
}

// containerPaths returns a list of all the available API paths that are available for a container.
func (a *Api) containerPaths(request *restful.Request, response *restful.Response) {
	entities := []string{
		"metrics/",
		"stats/",
	}
	response.WriteEntity(entities)
}

// allNodes returns a list of all the available node names in the model.
func (a *Api) allNodes(request *restful.Request, response *restful.Response) {
}

// allNamespaces returns a list of all the available namespaces in the model.
func (a *Api) allNamespaces(request *restful.Request, response *restful.Response) {
}

// allPods returns a list of all the available pods in the model.
func (a *Api) allPods(request *restful.Request, response *restful.Response) {
}

// allPodContainers returns a list of all the available pod containers in the model.
func (a *Api) allPodContainers(request *restful.Request, response *restful.Response) {
}

// allFreeContainers returns a list of all the available free containers in the cluster.
func (a *Api) allFreeContainers(request *restful.Request, response *restful.Response) {
}

// nodePods returns a list of all the available API paths that are available for a node.
func (a *Api) nodePods(request *restful.Request, response *restful.Response) {
}

// availableMetrics returns a list of available metric names.
// These metric names can be used to extract metrics from the various model entities.
func (a *Api) availableMetrics(request *restful.Request, response *restful.Response) {
}

// clusterStats returns a map of StatBundles for each usage metric of the Cluster entity.
func (a *Api) clusterStats(request *restful.Request, response *restful.Response) {
}

// clusterMetrics returns a metric timeseries for a metric of the Cluster entity.
func (a *Api) clusterMetrics(request *restful.Request, response *restful.Response) {
}

// nodeStats returns a map of StatBundles for each usage metric of a Node entity.
func (a *Api) nodeStats(request *restful.Request, response *restful.Response) {
}

// nodeMetrics returns a metric timeseries for a metric of the Node entity.
func (a *Api) nodeMetrics(request *restful.Request, response *restful.Response) {
}

// namespaceStats returns a map of StatBundles for each usage metric of a Namespace entity.
func (a *Api) namespaceStats(request *restful.Request, response *restful.Response) {
}

// namespaceMetrics returns a metric timeseries for a metric of the Namespace entity.
func (a *Api) namespaceMetrics(request *restful.Request, response *restful.Response) {
}

// podStats returns a map of StatBundles for each usage metric of a Pod entity.
func (a *Api) podStats(request *restful.Request, response *restful.Response) {
}

// podMetrics returns a metric timeseries for a metric of the Pod entity.
func (a *Api) podMetrics(request *restful.Request, response *restful.Response) {
}

func (a *Api) podListMetrics(request *restful.Request, response *restful.Response) {
}

// podContainerStats returns a map of StatBundles for each usage metric of a PodContainer entity.
func (a *Api) podContainerStats(request *restful.Request, response *restful.Response) {
}

// podContainerMetrics returns a metric timeseries for a metric of a Pod Container entity.
// podContainerMetrics uses the namespace-name/pod-name/container-name path.
func (a *Api) podContainerMetrics(request *restful.Request, response *restful.Response) {
}

// freeContainerStats returns a map of StatBundles for each usage metric of a free Container entity.
func (a *Api) freeContainerStats(request *restful.Request, response *restful.Response) {
}

// freeContainerMetrics returns a metric timeseries for a metric of the Container entity.
// freeContainerMetrics addresses only free containers, by using the node-name/container-name path.
func (a *Api) freeContainerMetrics(request *restful.Request, response *restful.Response) {
}
