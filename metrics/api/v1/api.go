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
	restful "github.com/emicklei/go-restful"

	"k8s.io/heapster/metrics/api/v1/types"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sinks/metric"
)

type Api struct {
	runningInKubernetes bool
	metricSink          *metricsink.MetricSink
	gkeMetrics          map[string]core.MetricDescriptor
	gkeLabels           map[string]core.LabelDescriptor
}

// Create a new Api to serve from the specified cache.
func NewApi(runningInKubernetes bool, metricSink *metricsink.MetricSink) *Api {
	gkeMetrics := make(map[string]core.MetricDescriptor)
	gkeLabels := make(map[string]core.LabelDescriptor)
	for _, val := range core.StandardMetrics {
		gkeMetrics[val.Name] = val.MetricDescriptor
	}
	for _, val := range core.CommonLabels() {
		gkeLabels[val.Key] = val
	}
	for _, val := range core.PodLabels() {
		gkeLabels[val.Key] = val
	}

	return &Api{
		runningInKubernetes: runningInKubernetes,
		metricSink:          metricSink,
		gkeMetrics:          gkeMetrics,
		gkeLabels:           gkeLabels,
	}
}

// Register the mainApi on the specified endpoint.
func (a *Api) Register(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/api/v1/metric-export").
		Doc("Exports the latest point for all Heapster metrics").
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("").
		To(a.exportMetrics).
		Doc("export the latest data point for all metrics").
		Operation("exportMetrics").
		Writes([]*types.Timeseries{}))
	container.Add(ws)
	ws = new(restful.WebService)
	ws.Path("/api/v1/metric-export-schema").
		Doc("Schema for metrics exported by heapster").
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("").
		To(a.exportMetricsSchema).
		Doc("export the schema for all metrics").
		Operation("exportmetricsSchema").
		Writes(types.TimeseriesSchema{}))
	container.Add(ws)

	if a.metricSink != nil {
		a.RegisterModel(container)
	}
}

func convertLabelDescriptor(ld core.LabelDescriptor) types.LabelDescriptor {
	return types.LabelDescriptor{
		Key:         ld.Key,
		Description: ld.Description,
	}
}

func convertMetricDescriptor(md core.MetricDescriptor) types.MetricDescriptor {
	result := types.MetricDescriptor{
		Name:        md.Name,
		Description: md.Description,
		Labels:      make([]types.LabelDescriptor, 0, len(md.Labels)),
	}
	for _, label := range md.Labels {
		result.Labels = append(result.Labels, convertLabelDescriptor(label))
	}

	switch md.Type {
	case core.MetricCumulative:
		result.Type = "cumulative"
	case core.MetricGauge:
		result.Type = "gauge"
	}

	switch md.ValueType {
	case core.ValueInt64:
		result.ValueType = "int64"
	case core.ValueFloat:
		result.ValueType = "double"
	}

	switch md.Units {
	case core.UnitsBytes:
		result.Units = "bytes"
	case core.UnitsMilliseconds:
		result.Units = "ms"
	case core.UnitsNanoseconds:
		result.Units = "ns"
	case core.UnitsMillicores:
		result.Units = "millicores"
	}
	return result
}

func (a *Api) exportMetricsSchema(request *restful.Request, response *restful.Response) {
	result := types.TimeseriesSchema{
		Metrics:      make([]types.MetricDescriptor, 0),
		CommonLabels: make([]types.LabelDescriptor, 0),
		PodLabels:    make([]types.LabelDescriptor, 0),
	}
	for _, metric := range core.StandardMetrics {
		if _, found := a.gkeMetrics[metric.Name]; found {
			result.Metrics = append(result.Metrics, convertMetricDescriptor(metric.MetricDescriptor))
		}
	}
	for _, label := range core.CommonLabels() {
		if _, found := a.gkeLabels[label.Key]; found {
			result.PodLabels = append(result.PodLabels, convertLabelDescriptor(label))
		}
	}
	for _, label := range core.PodLabels() {
		if _, found := a.gkeLabels[label.Key]; found {
			result.PodLabels = append(result.PodLabels, convertLabelDescriptor(label))
		}
	}
	response.WriteEntity(result)
}

func (a *Api) exportMetrics(request *restful.Request, response *restful.Response) {
	shortStorage := a.metricSink.GetShortStore()
	tsmap := make(map[string]*types.Timeseries)

	for _, batch := range shortStorage {
		for key, ms := range batch.MetricSets {
			ts := tsmap[key]

			msType := ms.Labels[core.LabelMetricSetType.Key]

			if msType != core.MetricSetTypeNode &&
				msType != core.MetricSetTypePod &&
				msType != core.MetricSetTypePodContainer &&
				msType != core.MetricSetTypeSystemContainer {
				continue
			}

			if ts == nil {
				ts = &types.Timeseries{
					Metrics: make(map[string][]types.Point),
					Labels:  make(map[string]string),
				}
				for labelName, labelValue := range ms.Labels {
					if _, ok := a.gkeLabels[labelName]; ok {
						ts.Labels[labelName] = labelValue
					}
				}
				if ms.Labels[core.LabelMetricSetType.Key] == core.MetricSetTypeNode {
					ts.Labels[core.LabelContainerName.Key] = "machine"
				}
				tsmap[key] = ts
			}
			for metricName, metricVal := range ms.MetricValues {
				if _, ok := a.gkeMetrics[metricName]; ok {
					points := ts.Metrics[metricName]
					if points == nil {
						points = make([]types.Point, 0, len(shortStorage))
					}
					point := types.Point{
						Start: batch.Timestamp,
						End:   batch.Timestamp,
					}
					if metricVal.ValueType == core.ValueInt64 {
						point.Value = &metricVal.IntValue
					} else if metricVal.ValueType == core.ValueFloat {
						point.Value = &metricVal.FloatValue
					} else {
						continue
					}
					points = append(points, point)
					ts.Metrics[metricName] = points
				}
			}
		}
	}
	timeseries := make([]*types.Timeseries, 0, len(tsmap))
	for _, ts := range tsmap {
		timeseries = append(timeseries, ts)
	}

	response.WriteEntity(timeseries)
}
