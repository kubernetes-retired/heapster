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
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/hawkular/hawkular-client-go/metrics"
	"k8s.io/heapster/extpoints"

	sink_api "k8s.io/heapster/sinks/api"
	kube_api "k8s.io/kubernetes/pkg/api"
)

const (
	unitsTag       string = "units"
	typeTag        string = "type"
	descriptionTag string = "_description"
	descriptorTag  string = "descriptor_name"
	groupTag       string = "group_id"
	separator      string = "/"
)

type hawkularSink struct {
	client  *metrics.Client
	models  map[string]metrics.MetricDefinition // Model definitions
	regLock sync.Mutex
	reg     map[string]*metrics.MetricDefinition // Real definitions

	uri *url.URL
}

// START: ExternalSink interface implementations

func (self *hawkularSink) Register(mds []sink_api.MetricDescriptor) error {
	self.regLock.Lock()
	defer self.regLock.Unlock()

	// Create model definitions based on the MetricDescriptors
	for _, md := range mds {
		hmd := self.descriptorToDefinition(&md)
		self.models[md.Name] = hmd
	}

	// Fetch currently known metrics from Hawkular-Metrics and cache them
	prev, err := self.client.Definitions(metrics.Gauge)
	if err != nil {
		return err
	}

	for _, p := range prev {
		// If no descriptorTag is found, this metric does not belong to Heapster
		if mk, found := p.Tags[descriptorTag]; found {
			model := self.models[mk]
			if !self.recent(p, &model) {
				if err := self.client.UpdateTags(metrics.Gauge, p.Id, p.Tags); err != nil {
					return err
				}
			}
			self.reg[p.Id] = p
		}
	}

	return nil
}

func (self *hawkularSink) Unregister(mds []sink_api.MetricDescriptor) error {
	self.regLock.Lock()
	defer self.regLock.Unlock()
	return self.init()
}

// Checks that stored definition is up to date with the model
func (self *hawkularSink) recent(live *metrics.MetricDefinition, model *metrics.MetricDefinition) bool {
	recent := true
	for k := range model.Tags {
		if v, found := live.Tags[k]; !found {
			// There's a label that wasn't in our stored definition
			live.Tags[k] = v
			recent = false
		}
	}

	return recent
}

// Transform the MetricDescriptor to a format used by Hawkular-Metrics
func (self *hawkularSink) descriptorToDefinition(md *sink_api.MetricDescriptor) metrics.MetricDefinition {
	tags := make(map[string]string)
	// Postfix description tags with _description
	for _, l := range md.Labels {
		if len(l.Description) > 0 {
			tags[l.Key+descriptionTag] = l.Description
		}
	}

	if len(md.Units.String()) > 0 {
		tags[unitsTag] = md.Units.String()
	}
	if len(md.Type.String()) > 0 {
		tags[typeTag] = md.Type.String()
	}

	tags[descriptorTag] = md.Name

	hmd := metrics.MetricDefinition{
		Id:   md.Name,
		Tags: tags,
	}

	return hmd
}

func (self *hawkularSink) groupName(p *sink_api.Point) string {
	n := []string{p.Labels[sink_api.LabelContainerName.Key], p.Name}
	return strings.Join(n, separator)
}

func (self *hawkularSink) idName(p *sink_api.Point) string {
	n := []string{p.Labels[sink_api.LabelContainerName.Key], p.Labels[sink_api.LabelPodId.Key], p.Name}
	return strings.Join(n, separator)
}

// Check that metrics tags are defined on the Hawkular server and if not,
// register the metric definition.
func (self *hawkularSink) registerIfNecessary(t *sink_api.Timeseries) error {
	key := self.idName(t.Point)

	self.regLock.Lock()
	defer self.regLock.Unlock()

	// If found, check it matches the current stored definition (could be old info from
	// the stored metrics cache for example)
	if _, found := self.reg[key]; !found {
		// Register the metric descriptor here..
		if md, f := self.models[t.MetricDescriptor.Name]; f {
			// Set tag values
			for k, v := range t.Point.Labels {
				md.Tags[k] = v
			}

			md.Tags[groupTag] = self.groupName(t.Point)
			md.Tags[descriptorTag] = t.MetricDescriptor.Name

			// Create metric, use updateTags instead of Create because we know it is unique
			if err := self.client.UpdateTags(metrics.Gauge, key, md.Tags); err != nil {
				// Log error and don't add this key to the lookup table
				glog.Errorf("Could not update tags: %s", err)
				return err
			}

			// Add to the lookup table
			self.reg[key] = &md
			glog.Infof("Registered new metric definition: %s", key)
		} else {
			return fmt.Errorf("Could not find definition model with name %s", t.MetricDescriptor.Name)
		}
	}
	// TODO Compare the definition tags and update if necessary? Quite expensive operation..

	return nil
}

func (self *hawkularSink) StoreTimeseries(ts []sink_api.Timeseries) error {
	if len(ts) > 0 {
		mhs := make([]metrics.MetricHeader, 0, len(ts))

		for _, t := range ts {
			self.registerIfNecessary(&t)

			if t.MetricDescriptor.ValueType == sink_api.ValueBool {
				// TODO: Model to availability type once we see some real world examples
				break
			}

			mH, err := self.pointToMetricHeader(&t)
			if err != nil {
				// One transformation error should not prevent the whole process
				glog.Errorf(err.Error())
				continue
			}

			mhs = append(mhs, *mH)
		}

		return self.client.Write(mhs)
	}
	return nil
}

// Converts Timeseries to metric structure used by the Hawkular
func (self *hawkularSink) pointToMetricHeader(t *sink_api.Timeseries) (*metrics.MetricHeader, error) {

	p := t.Point
	name := self.idName(p)

	value, err := metrics.ConvertToFloat64(p.Value)
	if err != nil {
		return nil, err
	}

	m := metrics.Datapoint{
		Value:     value,
		Timestamp: metrics.UnixMilli(p.End),
	}

	// At the moment all the values are converted to gauges.
	mh := &metrics.MetricHeader{
		Id:   name,
		Data: []metrics.Datapoint{m},
		Type: metrics.Gauge,
	}
	return mh, nil
}

func (self *hawkularSink) DebugInfo() string {
	info := fmt.Sprintf("Hawkular-Metrics Sink\n")

	self.regLock.Lock()
	defer self.regLock.Unlock()
	info += fmt.Sprintf("Known metrics: %d", len(self.reg))

	// TODO Add here statistics from the Hawkular-Metrics client instance
	return info
}

func (self *hawkularSink) StoreEvents(events []kube_api.Event) error {
	// TODO: Delegate to Fabric8 event storage (if available) until Hawkular has a solution?
	return nil
}

func (self *hawkularSink) Name() string {
	return "Hawkular-Metrics sink"
}

// END: ExternalSink

func init() {
	extpoints.SinkFactories.Register(NewHawkularSink, "hawkular")
}

func (self *hawkularSink) init() error {
	p := metrics.Parameters{
		Tenant: "heapster",
		Host:   self.uri.Host,
	}

	// Connection parameters
	if len(self.uri.Path) > 0 {
		p.Path = self.uri.Path
	}

	opts := self.uri.Query()

	if v, found := opts["tenant"]; found {
		p.Tenant = v[0]
	}

	c, err := metrics.NewHawkularClient(p)
	if err != nil {
		return err
	}

	self.client = c
	self.reg = make(map[string]*metrics.MetricDefinition)
	self.models = make(map[string]metrics.MetricDefinition)

	glog.Infof("Initialised Hawkular Sink with parameters %v", p)
	return nil
}

func NewHawkularSink(u *url.URL, _ extpoints.HeapsterConf) ([]sink_api.ExternalSink, error) {
	sink := &hawkularSink{
		uri: u,
	}
	if err := sink.init(); err != nil {
		return nil, err
	}
	return []sink_api.ExternalSink{sink}, nil
}
