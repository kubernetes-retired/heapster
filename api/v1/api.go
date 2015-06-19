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

	"github.com/GoogleCloudPlatform/heapster/manager"
	sinksApi "github.com/GoogleCloudPlatform/heapster/sinks/api/v1"
	"github.com/GoogleCloudPlatform/heapster/util"
	restful "github.com/emicklei/go-restful"
	"github.com/golang/glog"
)

type Api struct {
	manager manager.Manager
}

// Create a new Api to serve from the specified cache.
func NewApi(m manager.Manager) *Api {
	return &Api{
		manager: m,
	}
}

// Register the Api on the specified endpoint.
func (a *Api) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
		Path("/api/v1/metric-export").
		Doc("Exports the latest point for all Heapster metrics").
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("").
		Filter(compressionFilter).
		To(a.exportMetrics).
		Doc("export the latest data point for all metrics").
		Operation("exportMetrics").
		Writes([]*Timeseries{}))
	container.Add(ws)
	ws = new(restful.WebService)
	ws.Path("/api/v1/metric-export-schema").
		Doc("Schema for metrics exported by heapster").
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("").
		To(a.exportMetricsSchema).
		Doc("export the schema for all metrics").
		Operation("exportmetricsSchema").
		Writes(TimeseriesSchema{}))
	container.Add(ws)
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

// Labels used by the target schema. A target schema uniquely identifies a container.
var targetLabelNames = map[string]struct{}{
	sinksApi.LabelPodId.Key:           struct{}{},
	sinksApi.LabelPodName.Key:         struct{}{},
	sinksApi.LabelPodNamespace.Key:    struct{}{},
	sinksApi.LabelContainerName.Key:   struct{}{},
	sinksApi.LabelLabels.Key:          struct{}{},
	sinksApi.LabelHostname.Key:        struct{}{},
	sinksApi.LabelHostID.Key:          struct{}{},
	sinksApi.LabelPodNamespaceUID.Key: struct{}{},
}

// Separates target schema labels from other labels.
func separateLabels(labels map[string]string) (map[string]string, map[string]string) {
	targetLabels := make(map[string]string, len(targetLabelNames))
	otherLabels := make(map[string]string, len(labels)-len(targetLabels))
	for label, _ := range labels {
		// Ignore blank labels.
		if label == "" {
			continue
		}

		if _, ok := targetLabelNames[label]; ok {
			targetLabels[label] = labels[label]
		} else {
			otherLabels[label] = labels[label]
		}
	}

	return targetLabels, otherLabels
}

func (a *Api) exportMetricsSchema(request *restful.Request, response *restful.Response) {
	result := TimeseriesSchema{}
	for _, label := range sinksApi.CommonLabels() {
		result.CommonLabels = append(result.CommonLabels, LabelDescriptor{
			Key:         label.Key,
			Description: label.Description,
		})
	}
	for _, label := range sinksApi.PodLabels() {
		result.PodLabels = append(result.PodLabels, LabelDescriptor{
			Key:         label.Key,
			Description: label.Description,
		})
	}

	for _, metric := range sinksApi.SupportedStatMetrics() {
		md := MetricDescriptor{
			Name:        metric.Name,
			Description: metric.Description,
			Type:        metric.Type.String(),
			ValueType:   metric.ValueType.String(),
			Units:       metric.Units.String(),
		}
		for _, label := range metric.Labels {
			md.Labels = append(md.Labels, LabelDescriptor{
				Key:         label.Key,
				Description: label.Description,
			})
		}
		result.Metrics = append(result.Metrics, md)
	}
	response.WriteEntity(result)
}

func (a *Api) exportMetrics(request *restful.Request, response *restful.Response) {
	points, err := a.manager.ExportMetrics()
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	// Group points by target labels.
	timeseriesForTargetLabels := map[string]*Timeseries{}
	for _, point := range points {
		targetLabels, otherLabels := separateLabels(point.Labels)
		labelsStr := util.LabelsToString(targetLabels, ",")

		// Add timeseries if it does not exist.
		timeseries, ok := timeseriesForTargetLabels[labelsStr]
		if !ok {
			timeseries = &Timeseries{
				Metrics: map[string][]Point{},
				Labels:  targetLabels,
			}
			timeseriesForTargetLabels[labelsStr] = timeseries
		}

		// Add point to this timeseries
		timeseries.Metrics[point.Name] = append(timeseries.Metrics[point.Name], Point{
			Start:  point.Start,
			End:    point.End,
			Labels: otherLabels,
			Value:  point.Value,
		})
	}

	// Turn into a slice.
	timeseries := make([]*Timeseries, 0, len(timeseriesForTargetLabels))
	for _, val := range timeseriesForTargetLabels {
		timeseries = append(timeseries, val)
	}

	response.WriteEntity(timeseries)
}
