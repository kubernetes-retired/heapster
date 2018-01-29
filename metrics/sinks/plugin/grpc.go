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
	ptype "github.com/golang/protobuf/ptypes"
	context "golang.org/x/net/context"
	"k8s.io/heapster/metrics/core"
	proto "k8s.io/heapster/metrics/sinks/plugin/proto"
)

type GRPCClient struct{ client proto.MetricPluginClient }

func (m *GRPCClient) WriteMetricsPoint(point *PluginSinkPoint) error {
	timestamp, err := ptype.TimestampProto(point.MetricsTimestamp)
	if err != nil {
		return err
	}
	_, err = m.client.WriteMetricsPoint(context.Background(), &proto.MetricPoint{
		Name: point.MetricsName,
		Value: &proto.MetricValue{
			IntValue:   point.MetricsValue.IntValue,
			FloatValue: point.MetricsValue.FloatValue,
			MetricType: point.MetricsValue.MetricType.String(),
			ValueType:  point.MetricsValue.ValueType.String(),
		},
		Timestamp: timestamp,
		Tags:      point.MetricsTags,
	})
	return err
}

type GRPCServer struct {
	Impl MetricSink
}

func getValueType(valueType string) core.ValueType {
	switch valueType {
	case "int64":
		return core.ValueInt64
	case "double":
		return core.ValueFloat
	default:
		return -1
	}
}

func getMetricType(metricType string) core.MetricType {
	switch metricType {
	case "cumulative":
		return core.MetricCumulative
	case "gauge":
		return core.MetricGauge
	case "delta":
		return core.MetricDelta
	default:
		return -1
	}
}

func (m *GRPCServer) WriteMetricsPoint(ctx context.Context, req *proto.MetricPoint) (*proto.Empty, error) {
	timestamp, err := ptype.Timestamp(req.Timestamp)
	if err != nil {
		return nil, err
	}
	return &proto.Empty{}, m.Impl.WriteMetricsPoint(&PluginSinkPoint{
		MetricsName:      req.Name,
		MetricsTimestamp: timestamp,
		MetricsValue: core.MetricValue{
			IntValue:   req.Value.IntValue,
			FloatValue: req.Value.FloatValue,
			ValueType:  getValueType(req.Value.ValueType),
			MetricType: getMetricType(req.Value.MetricType),
		},
		MetricsTags: req.Tags,
	})
}
