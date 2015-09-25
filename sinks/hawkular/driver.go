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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/hawkular/hawkular-client-go/metrics"
	"k8s.io/heapster/extpoints"

	sink_api "k8s.io/heapster/sinks/api"
	kube_api "k8s.io/kubernetes/pkg/api"
	kube_client "k8s.io/kubernetes/pkg/client/unversioned"
	kubeClientCmd "k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

const (
	unitsTag       string = "units"
	typeTag        string = "type"
	descriptionTag string = "_description"
	descriptorTag  string = "descriptor_name"
	groupTag       string = "group_id"
	separator      string = "/"

	defaultServiceAccountFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

type hawkularSink struct {
	client  *metrics.Client
	models  map[string]*metrics.MetricDefinition // Model definitions
	regLock sync.Mutex
	reg     map[string]*metrics.MetricDefinition // Real definitions

	uri *url.URL

	labelTenant string
}

// START: ExternalSink interface implementations

func (self *hawkularSink) Register(mds []sink_api.MetricDescriptor) error {
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

// Fetches definitions from the server and checks that they're matching the descriptors
func (self *hawkularSink) updateDefinitions(mt metrics.MetricType) error {
	mds, err := self.client.Definitions(metrics.Filters(metrics.TypeFilter(mt)))
	if err != nil {
		return err
	}

	self.regLock.Lock()
	defer self.regLock.Unlock()

	for _, p := range mds {
		// If no descriptorTag is found, this metric does not belong to Heapster
		if mk, found := p.Tags[descriptorTag]; found {
			if model, f := self.models[mk]; f {
				if !self.recent(p, model) {
					if err := self.client.UpdateTags(mt, p.Id, p.Tags); err != nil {
						return err
					}
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

	tags[descriptorTag] = md.Name

	hmd := metrics.MetricDefinition{
		Id:   md.Name,
		Tags: tags,
		Type: heapsterTypeToHawkularType(md.Type),
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
func (self *hawkularSink) registerIfNecessary(t *sink_api.Timeseries, m ...metrics.Modifier) error {
	key := self.idName(t.Point)

	self.regLock.Lock()
	defer self.regLock.Unlock()

	// If found, check it matches the current stored definition (could be old info from
	// the stored metrics cache for example)
	if _, found := self.reg[key]; !found {
		// Register the metric descriptor here..
		if md, f := self.models[t.MetricDescriptor.Name]; f {
			// Copy the original map
			mdd := *md
			tags := make(map[string]string)
			for k, v := range mdd.Tags {
				tags[k] = v
			}
			mdd.Tags = tags

			// Set tag values
			for k, v := range t.Point.Labels {
				mdd.Tags[k] = v
			}

			mdd.Tags[groupTag] = self.groupName(t.Point)
			mdd.Tags[descriptorTag] = t.MetricDescriptor.Name

			// Create metric, use updateTags instead of Create because we know it is unique
			if err := self.client.UpdateTags(mdd.Type, key, mdd.Tags, m...); err != nil {
				// Log error and don't add this key to the lookup table
				glog.Errorf("Could not update tags: %s", err)
				return err
			}

			// Add to the lookup table
			self.reg[key] = &mdd
		} else {
			return fmt.Errorf("Could not find definition model with name %s", t.MetricDescriptor.Name)
		}
	}
	// TODO Compare the definition tags and update if necessary? Quite expensive operation..

	return nil
}

func (self *hawkularSink) StoreTimeseries(ts []sink_api.Timeseries) error {
	if len(ts) > 0 {
		tmhs := make(map[string][]metrics.MetricHeader)

		if &self.labelTenant == nil {
			tmhs[self.client.Tenant] = make([]metrics.MetricHeader, 0, len(ts))
		}

		wg := &sync.WaitGroup{}

		for _, t := range ts {
			t := t

			tenant := self.client.Tenant

			if &self.labelTenant != nil {
				if v, found := t.Point.Labels[self.labelTenant]; found {
					tenant = v
				}
			}

			// Registering should not block the processing
			wg.Add(1)
			go func(t *sink_api.Timeseries, tenant string) {
				defer wg.Done()
				self.registerIfNecessary(t, metrics.Tenant(tenant))
			}(&t, tenant)

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

			if _, found := tmhs[tenant]; !found {
				tmhs[tenant] = make([]metrics.MetricHeader, 0)
			}

			tmhs[tenant] = append(tmhs[tenant], *mH)
		}

		for k, v := range tmhs {
			wg.Add(1)
			go func(v []metrics.MetricHeader, k string) {
				defer wg.Done()
				if err := self.client.Write(v, metrics.Tenant(k)); err != nil {
					glog.Errorf(err.Error())
				}
			}(v, k)
		}
		wg.Wait()
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

	mh := &metrics.MetricHeader{
		Id:   name,
		Data: []metrics.Datapoint{m},
		Type: heapsterTypeToHawkularType(t.MetricDescriptor.Type),
	}

	return mh, nil
}

func heapsterTypeToHawkularType(t sink_api.MetricType) metrics.MetricType {
	switch t {
	case sink_api.MetricCumulative:
		return metrics.Counter
	case sink_api.MetricGauge:
		return metrics.Gauge
	default:
		return metrics.Gauge
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

func (self *hawkularSink) StoreEvents(events []kube_api.Event) error {
	return nil
}

func (self *hawkularSink) Name() string {
	return "Hawkular-Metrics Sink"
}

// END: ExternalSink

func init() {
	extpoints.SinkFactories.Register(NewHawkularSink, "hawkular")
}

func (self *hawkularSink) init() error {
	p := metrics.Parameters{
		Tenant: "heapster",
		Url:    self.uri.String(),
	}

	opts := self.uri.Query()

	if v, found := opts["tenant"]; found {
		p.Tenant = v[0]
	}

	if v, found := opts["labelToTenant"]; found {
		self.labelTenant = v[0]
	}

	if v, found := opts["useServiceAccount"]; found {
		if b, _ := strconv.ParseBool(v[0]); b {
			// If a readable service account token exists, then use it
			if contents, err := ioutil.ReadFile(defaultServiceAccountFile); err == nil {
				p.Token = string(contents)
			}
		}
	}

	// Authentication / Authorization parameters
	tC := &tls.Config{}

	if v, found := opts["auth"]; found {
		if _, f := opts["caCert"]; f {
			return fmt.Errorf("Both auth and caCert files provided, combination is not supported")
		}
		if len(v[0]) > 0 {
			// Authfile
			kubeConfig, err := kubeClientCmd.NewNonInteractiveDeferredLoadingClientConfig(&kubeClientCmd.ClientConfigLoadingRules{
				ExplicitPath: v[0]},
				&kubeClientCmd.ConfigOverrides{}).ClientConfig()
			if err != nil {
				return err
			}
			tC, err = kube_client.TLSConfigFor(kubeConfig)
			if err != nil {
				return err
			}
		}
	}

	if v, found := opts["caCert"]; found {
		caCert, err := ioutil.ReadFile(v[0])
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tC.RootCAs = caCertPool
	}

	if v, found := opts["insecure"]; found {
		_, f := opts["caCert"]
		_, f2 := opts["auth"]
		if f || f2 {
			return fmt.Errorf("Insecure can't be defined with auth or caCert")
		}
		insecure, err := strconv.ParseBool(v[0])
		if err != nil {
			return err
		}
		tC.InsecureSkipVerify = insecure
	}

	p.TLSConfig = tC

	c, err := metrics.NewHawkularClient(p)
	if err != nil {
		return err
	}

	self.client = c
	self.reg = make(map[string]*metrics.MetricDefinition)
	self.models = make(map[string]*metrics.MetricDefinition)

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
