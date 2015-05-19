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

package riemann

import (
	riemann_api "github.com/bigdatadev/goryman"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockClient struct {
	connected bool
	closed    bool
}

func (self *mockClient) Connect() error                     { self.connected = true; return nil }
func (self *mockClient) Close() error                       { self.closed = true; return nil }
func (self *mockClient) SendEvent(*riemann_api.Event) error { return nil }
func getMock() riemannClient                                { return &mockClient{} }

var _ = Describe("Driver", func() {
	Describe("new", func() {
		It("creates a new Riemann sink pointing to the requested instance", func() {
			var _, err = new(getMock, riemannConfig{})
			Expect(err).To(BeNil())
		})
		// func new(addr string) (sink_api.ExternalSink, error) { return nil, nil }
	})

	Describe("Name", func() {
		It("gives a user-friendly string describing the sink", func() {
			var subject, err = new(getMock, riemannConfig{})
			Expect(err).To(BeNil())

			var name = subject.Name()

			Expect(name).To(Equal("Riemann"))
		})
	})

	Describe("Register", func() {
		// func (self *riemannSink) Register(descriptor []sink_api.MetricDescriptor) error { return nil }
		It("registers a metric with Riemann (no-op)", func() {})
	})

	PDescribe("StoreTimeseries", func() {
		// func (self *riemannSink) StoreTimeseries(ts []sink_api.Timeseries) error { return nil }
		It("sends a collection of Timeseries to Riemann", func() {})
	})

	PDescribe("StoreEvents", func() {
		// func (self *riemannSink) StoreEvents(event []kube_api.Event) error { return nil }
		It("sends a collection of Kubernetes Events to Riemann", func() {})
	})

	Describe("DebugInfo", func() {
		// func (self *riemannSink) DebugInfo() string { return "" }
		It("gives debug information specific to Riemann", func() {
			var subject, err = new(getMock, riemannConfig{})
			Expect(err).To(BeNil())
			var debugInfo = subject.DebugInfo()
			Expect(debugInfo).ToNot(Equal(""))
		})
	})
})
