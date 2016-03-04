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

package monasca

import (
	"time"

	"github.com/rackspace/gophercloud/openstack/identity/v3/tokens"
	sinksApi "k8s.io/heapster/sinks/api"
)

var measureStart = time.Now()
var measureEnd = time.Unix(measureStart.Unix()+20, 0)

var testInput = []sinksApi.Timeseries{
	sinksApi.Timeseries{
		Point: &sinksApi.Point{
			Name: "invalid",
		}},
	sinksApi.Timeseries{
		Point: &sinksApi.Point{
			Name: "m1",
			Labels: map[string]string{
				sinksApi.LabelHostname.Key: "h1",
			},
			Start: measureStart,
			End:   measureEnd,
			Value: false,
		},
		MetricDescriptor: &sinksApi.MetricDescriptor{
			Name: "m1",
			Labels: []sinksApi.LabelDescriptor{
				sinksApi.LabelHostname,
			},
			Type:      sinksApi.MetricCumulative,
			ValueType: sinksApi.ValueBool,
			Units:     sinksApi.UnitsMilliseconds,
		}},
	sinksApi.Timeseries{
		Point: &sinksApi.Point{
			Name: "m2",
			Labels: map[string]string{
				sinksApi.LabelHostname.Key: "h2",
			},
			Start: measureStart,
			End:   measureEnd,
			Value: -1023.0233,
		},
		MetricDescriptor: &sinksApi.MetricDescriptor{
			Name: "m2",
			Labels: []sinksApi.LabelDescriptor{
				sinksApi.LabelHostname,
			},
			Type:      sinksApi.MetricGauge,
			ValueType: sinksApi.ValueDouble,
			Units:     sinksApi.UnitsMilliseconds,
		}},
	sinksApi.Timeseries{
		Point: &sinksApi.Point{
			Name: "m3",
			Labels: map[string]string{
				sinksApi.LabelHostname.Key: "h3",
			},
			Start: measureStart,
			End:   measureEnd,
			Value: 2 ^ 63,
		},
		MetricDescriptor: &sinksApi.MetricDescriptor{
			Name: "m3",
			Labels: []sinksApi.LabelDescriptor{
				sinksApi.LabelHostname,
			},
			Type:      sinksApi.MetricGauge,
			ValueType: sinksApi.ValueInt64,
			Units:     sinksApi.UnitsMilliseconds,
		}},
	sinksApi.Timeseries{
		Point: &sinksApi.Point{
			Name: "cpu/usage",
			Labels: map[string]string{
				sinksApi.LabelHostname.Key:      "10.140.32.11",
				sinksApi.LabelContainerName.Key: "POD",
				sinksApi.LabelPodName.Key:       "mypod-hc3s",
				sinksApi.LabelLabels.Key:        "run:test,pod.name:default/test-u2dc",
				sinksApi.LabelHostID.Key:        "",
			},
			Start: measureStart,
			End:   measureEnd,
			Value: true,
		},
		MetricDescriptor: &sinksApi.MetricDescriptor{
			Name:        "cpu/usage",
			Description: "foo",
			Labels: []sinksApi.LabelDescriptor{
				sinksApi.LabelHostname,
				sinksApi.LabelContainerName,
				sinksApi.LabelLabels,
				sinksApi.LabelHostID,
				sinksApi.LabelPodName,
			},
			Type:      sinksApi.MetricCumulative,
			ValueType: sinksApi.ValueBool,
			Units:     sinksApi.UnitsMilliseconds,
		}},
	sinksApi.Timeseries{
		Point: &sinksApi.Point{
			Name: "memory/usage",
			Labels: map[string]string{
				sinksApi.LabelHostname.Key:      "10.140.32.15",
				sinksApi.LabelContainerName.Key: "machine",
				sinksApi.LabelLabels.Key:        "pod.name:default/test-u2dc,run:test2,foo:bar",
				sinksApi.LabelHostID.Key:        "myhost",
			},
			Start: measureStart,
			End:   measureEnd,
			Value: 64,
		},
		MetricDescriptor: &sinksApi.MetricDescriptor{
			Name:        "memory/usage",
			Description: "bar",
			Labels: []sinksApi.LabelDescriptor{
				sinksApi.LabelHostname,
				sinksApi.LabelContainerName,
				sinksApi.LabelLabels,
				sinksApi.LabelHostID,
			},
			Type:      sinksApi.MetricGauge,
			ValueType: sinksApi.ValueInt64,
			Units:     sinksApi.UnitsMilliseconds,
		}},
}

var expectedTransformed = []metric{
	metric{
		Name: "m1",
		Dimensions: map[string]string{
			"component":                     emptyValue,
			"hostname":                      "h1",
			"service":                       "kubernetes",
			sinksApi.LabelContainerName.Key: emptyValue,
		},
		Value:     0.0,
		Timestamp: measureEnd.UnixNano() / 1000000,
		ValueMeta: map[string]string{},
	},
	metric{
		Name: "m2",
		Dimensions: map[string]string{
			"component":                     emptyValue,
			"hostname":                      "h2",
			"service":                       "kubernetes",
			sinksApi.LabelContainerName.Key: emptyValue,
		},
		Value:     -1023.0233,
		Timestamp: measureEnd.UnixNano() / 1000000,
		ValueMeta: map[string]string{},
	},
	metric{
		Name: "m3",
		Dimensions: map[string]string{
			"component":                     emptyValue,
			"hostname":                      "h3",
			"service":                       "kubernetes",
			sinksApi.LabelContainerName.Key: emptyValue,
		},
		Value:     2 ^ 63,
		Timestamp: measureEnd.UnixNano() / 1000000,
		ValueMeta: map[string]string{},
	},
	metric{
		Name: "cpu.usage",
		Dimensions: map[string]string{
			"component":                     "mypod-hc3s",
			"hostname":                      "10.140.32.11",
			"service":                       "kubernetes",
			sinksApi.LabelContainerName.Key: "POD",
		},
		Value:     1.0,
		Timestamp: measureEnd.UnixNano() / 1000000,
		ValueMeta: map[string]string{
			sinksApi.LabelLabels.Key: "run:test pod.name:default/test-u2dc",
		},
	},
	metric{
		Name: "memory.usage",
		Dimensions: map[string]string{
			"component":                     emptyValue,
			"hostname":                      "10.140.32.15",
			"service":                       "kubernetes",
			sinksApi.LabelContainerName.Key: "machine",
		},
		Value:     64.0,
		Timestamp: measureEnd.UnixNano() / 1000000,
		ValueMeta: map[string]string{
			sinksApi.LabelLabels.Key: "pod.name:default/test-u2dc run:test2 foo:bar",
			sinksApi.LabelHostID.Key: "myhost",
		},
	}}

const testToken = "e80b74"

var invalidToken = &tokens.Token{ID: "invalidToken", ExpiresAt: time.Unix(time.Now().Unix()-5000, 0)}
var validToken = &tokens.Token{ID: testToken, ExpiresAt: time.Unix(time.Now().Unix()+50000, 0)}

var testConfig = Config{}

const (
	testUsername   = "Joe"
	testPassword   = "bar"
	testUserID     = "0ca8f6"
	testDomainID   = "1789d1"
	testDomainName = "example.com"
)

var (
	ksVersionResp       string
	ksAuthResp          string
	ksServicesResp      string
	ksEndpointsResp     string
	monUnauthorizedResp string
	monEmptyDimResp     string
)

func initKeystoneRespStubs() {
	ksVersionResp = `{
                      "versions": {
                        "values": [{
                          "status": "stable",
                          "updated": "2015-03-30T00:00:00Z",
                          "id": "v3.4",
                          "links": [{
                            "href": "` + keystoneAPIStub.URL + `",
                            "rel": "self"
                          }]
                        }]
                      }
                    }`
	ksAuthResp = `{
                    "token": {
                        "audit_ids": ["VcxU2JYqT8OzfUVvrjEITQ", "qNUTIJntTzO1-XUk5STybw"],
                        "expires_at": "2013-02-27T18:30:59.999999Z",
                        "issued_at": "2013-02-27T16:30:59.999999Z",
                        "methods": [
                            "password"
                        ],
                        "user": {
                            "domain": {
                                "id": "1789d1",
                                "name": "example.com"
                            },
                            "id": "0ca8f6",
                            "name": "Joe"
                        }
                    }
                }`
	ksServicesResp = `{
                        "services": [{
                          "description": "Monasca Service",
                          "id": "ee057c",
                          "links": {
                            "self": "` + keystoneAPIStub.URL + `/v3/services/ee057c"
                          },
                          "name": "Monasca",
                          "type": "monitoring"
                        }],
                        "links": {
                          "self": "` + keystoneAPIStub.URL + `/v3/services",
                          "previous": null,
                          "next": null
                        }
                    }`
	ksEndpointsResp = `{
                        "endpoints": [
                            {
                                "enabled": true,
                                "id": "6fedc0",
                                "interface": "public",
                                "links": {
                                    "self": "` + keystoneAPIStub.URL + `/v3/endpoints/6fedc0"
                                },
                                "region_id": "us-east-1",
                                "service_id": "ee057c",
                                "url": "` + monascaAPIStub.URL + `"
                            }
                        ],
                        "links": {
                            "self": "` + keystoneAPIStub.URL + `/v3/endpoints",
                            "previous": null,
                            "next": null
                        }
                    }`
	monUnauthorizedResp = "Invaild token provided"
	monEmptyDimResp = "Empty dimension detected"
}
