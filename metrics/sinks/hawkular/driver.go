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

package hawkular

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/golang/glog"
	"github.com/hawkular/hawkular-client-go/metrics"

	"k8s.io/heapster/metrics/core"
)

const (
	unitsTag       = "units"
	descriptionTag = "_description"
	descriptorTag  = "descriptor_name"
	groupTag       = "group_id"
	separator      = "/"

	defaultServiceAccountFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// START: ExternalSink interface implementations

func (self *hawkularSink) Register(mds []core.MetricDescriptor) error {
	// Create model definitions based on the MetricDescriptors
	for _, md := range mds {
		hmd := self.descriptorToDefinition(&md)
		self.models[md.Name] = &hmd
	}

	// Fetch currently known metrics from Hawkular-Metrics and cache them
	types := []metrics.MetricType{metrics.Gauge, metrics.Counter}
	for _, t := range types {
		err := self.updateDefinitions(t)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *hawkularSink) Stop() {
	self.regLock.Lock()
	defer self.regLock.Unlock()
	self.init()
}

func (self *hawkularSink) ExportData(db *core.DataBatch) {
	totalCount := 0
	for _, ms := range db.MetricSets {
		totalCount += len(ms.MetricValues)
	}

	// TODO: !!!! Limit number of metrics per batch !!!!
	if len(db.MetricSets) > 0 {
		tmhs := make(map[string][]metrics.MetricHeader)

		if &self.labelTenant == nil {
			tmhs[self.client.Tenant] = make([]metrics.MetricHeader, 0, totalCount)
		}

		wg := &sync.WaitGroup{}

		for _, ms := range db.MetricSets {
		Store:
			for metricName := range ms.MetricValues {

				for _, filter := range self.filters {
					if !filter(ms, metricName) {
						continue Store
					}
				}

				tenant := self.client.Tenant

				if &self.labelTenant != nil {
					if v, found := ms.Labels[self.labelTenant]; found {
						tenant = v
					}
				}

				// Registering should not block the processing
				wg.Add(1)
				go func(ms *core.MetricSet, metricName string, tenant string) {
					defer wg.Done()
					self.registerIfNecessary(ms, metricName, metrics.Tenant(tenant))
				}(ms, metricName, tenant)

				mH, err := self.pointToMetricHeader(ms, metricName, db.Timestamp)
				if err != nil {
					// One transformation error should not prevent the whole process
					glog.Errorf(err.Error())
					continue
				}

				if _, found := tmhs[tenant]; !found {
					tmhs[tenant] = make([]metrics.MetricHeader, 0)
				}

				tmhs[tenant] = append(tmhs[tenant], *mH)
			}
		}

		for k, v := range tmhs {
			wg.Add(1)
			go func(v []metrics.MetricHeader, k string) {
				defer wg.Done()
				m := make([]metrics.Modifier, len(self.modifiers), len(self.modifiers)+1)
				copy(m, self.modifiers)
				m = append(m, metrics.Tenant(k))
				if err := self.client.Write(v, m...); err != nil {
					glog.Errorf(err.Error())
				}
			}(v, k)
		}
		wg.Wait()
	}
}

func (self *hawkularSink) DebugInfo() string {
	info := fmt.Sprintf("%s\n", self.Name())

	self.regLock.Lock()
	defer self.regLock.Unlock()
	info += fmt.Sprintf("Known metrics: %d\n", len(self.reg))
	if &self.labelTenant != nil {
		info += fmt.Sprintf("Using label '%s' as tenant information\n", self.labelTenant)
	}

	// TODO Add here statistics from the Hawkular-Metrics client instance
	return info
}

func (self *hawkularSink) Name() string {
	return "Hawkular-Metrics Sink"
}

func NewHawkularSink(u *url.URL) (core.DataSink, error) {
	sink := &hawkularSink{
		uri: u,
	}
	if err := sink.init(); err != nil {
		return nil, err
	}
	metrics := make([]core.MetricDescriptor, 0, len(core.StandardMetrics))
	for _, metric := range core.StandardMetrics {
		metrics = append(metrics, metric.MetricDescriptor)
	}
	sink.Register(metrics)
	return sink, nil
}
