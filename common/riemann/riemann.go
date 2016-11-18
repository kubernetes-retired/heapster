// Copyright 2016 Google Inc. All Rights Reserved.
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
	"github.com/golang/glog"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"sync"
)

type RiemannClient interface {
	Connect() error
	Close() error
	SendEvent(e *riemann_api.Event) error
}

type RiemannConfig struct {
	Host  string
	Ttl   float32
	State string
	Tags  []string
}

type RiemannSink struct {
	Client RiemannClient
	Config RiemannConfig
	sync.RWMutex
}

const (
	// Maximum number of riemann Events to be sent in one batch.
	MaxSendBatchSize = 10000
	MaxRetries       = 2
)

func CreateRiemannSink(uri *url.URL) (*RiemannSink, error) {
	c := RiemannConfig{
		Host:  "riemann-heapster:5555",
		Ttl:   60.0,
		State: "",
		Tags:  make([]string, 0),
	}
	if len(uri.Host) > 0 {
		c.Host = uri.Host
	}
	options := uri.Query()
	if len(options["ttl"]) > 0 {
		var ttl, err = strconv.ParseFloat(options["ttl"][0], 32)
		if err != nil {
			return nil, err
		}
		c.Ttl = float32(ttl)
	}
	if len(options["state"]) > 0 {
		c.State = options["state"][0]
	}
	if len(options["tags"]) > 0 {
		c.Tags = options["tags"]
	}

	glog.Infof("Riemann sink URI: '%+v', host: '%+v', options: '%+v', ", uri, c.Host, options)
	rs := &RiemannSink{
		Client: nil,
		Config: c,
	}

	client, err := SetupRiemannClient(c)
	if err != nil {
		glog.Warningf("Riemann sink not connected: %v", err)
		// Warn but return the sink.
	} else {
		rs.Client = client
	}
	return rs, nil
}

func SetupRiemannClient(config RiemannConfig) (RiemannClient, error) {
	client := riemann_api.NewGorymanClient(config.Host)
	runtime.SetFinalizer(client, func(c RiemannClient) { c.Close() })
	err := client.Connect()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func RiemannValue(value interface{}) interface{} {
	// Workaround for error from goryman: "Metric of invalid type (type int64)"
	if reflect.TypeOf(value).Kind() == reflect.Int64 {
		return int(value.(int64))
	}
	return value
}
