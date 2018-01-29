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
	proto "k8s.io/heapster/events/sinks/plugin/proto"
)

type GRPCClient struct{ client proto.EventPluginClient }

func (m *GRPCClient) WriteEventsPoint(point *PluginSinkPoint) error {
	timestamp, err := ptype.TimestampProto(point.EventTimestamp)
	if err != nil {
		return err
	}
	_, err = m.client.WriteEventsPoint(context.Background(), &proto.EventPoint{
		Event:     point.EventValue,
		Timestamp: timestamp,
		Tags:      point.EventTags,
	})
	return err
}

type GRPCServer struct {
	Impl EventSink
}

func (m *GRPCServer) WriteEventsPoint(ctx context.Context, req *proto.EventPoint) (*proto.Empty, error) {
	timestamp, err := ptype.Timestamp(req.Timestamp)
	if err != nil {
		return nil, err
	}
	return &proto.Empty{}, m.Impl.WriteEventsPoint(&PluginSinkPoint{
		EventValue:     req.Event,
		EventTimestamp: timestamp,
		EventTags:      req.Tags,
	})
}
