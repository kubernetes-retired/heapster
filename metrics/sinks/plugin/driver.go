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

package plugin

import (
	"net/url"
	"sync"
	"time"

	"context"

	"github.com/golang/glog"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	plugin_common "k8s.io/heapster/common/plugin"
	"k8s.io/heapster/metrics/core"
	proto "k8s.io/heapster/metrics/sinks/plugin/proto"
)

func init() {
	plugin_common.PluginMap["metrics"] = &MetricSinkPlugin{}
}

type PluginSinkPoint struct {
	MetricsName      string
	MetricsValue     core.MetricValue
	MetricsTimestamp time.Time
	MetricsTags      map[string]string
}

type MetricSink interface {
	WriteMetricsPoint(point *PluginSinkPoint) error
}

type MetricSinkPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl MetricSink
}

func (p *MetricSinkPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterMetricPluginServer(s, &GRPCServer{
		Impl: p.Impl,
	})
	return nil
}

func (p *MetricSinkPlugin) GRPCClient(context context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{
		client: proto.NewMetricPluginClient(c),
	}, nil
}

type GoPluginSink struct {
	plugin_common.GoPluginSink
	MetricSink
	sync.RWMutex
}

func (sink *GoPluginSink) ExportData(dataBatch *core.DataBatch) {
	sink.Lock()
	defer sink.Unlock()

	for _, metricSet := range dataBatch.MetricSets {
		for metricName, metricValue := range metricSet.MetricValues {
			point := &PluginSinkPoint{
				MetricsName:      metricName,
				MetricsTags:      metricSet.Labels,
				MetricsValue:     metricValue,
				MetricsTimestamp: dataBatch.Timestamp.UTC(),
			}
			err := sink.WriteMetricsPoint(point)
			if err != nil {
				glog.Errorf("Failed to produce metric message: %s", err)
			}
		}
		for _, metric := range metricSet.LabeledMetrics {
			labels := make(map[string]string)
			for k, v := range metricSet.Labels {
				labels[k] = v
			}
			for k, v := range metric.Labels {
				labels[k] = v
			}
			point := &PluginSinkPoint{
				MetricsName:      metric.Name,
				MetricsTags:      labels,
				MetricsValue:     metric.MetricValue,
				MetricsTimestamp: dataBatch.Timestamp.UTC(),
			}
			err := sink.WriteMetricsPoint(point)
			if err != nil {
				glog.Errorf("Failed to produce metric message: %s", err)
			}
		}
	}
}

func NewGoPluginSink(uri *url.URL) (core.DataSink, error) {
	client, err := plugin_common.NewGoPluginClient(uri)
	if err != nil {
		return nil, err
	}
	sink, err := client.Plugin("metrics")
	if err != nil {
		return nil, err
	}
	return &GoPluginSink{
		GoPluginSink: client,
		MetricSink:   sink.(MetricSink),
	}, nil
}

var _ plugin.GRPCPlugin = &MetricSinkPlugin{}
