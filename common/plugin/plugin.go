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
	"fmt"
	"net/url"
	"os/exec"

	plugin "github.com/hashicorp/go-plugin"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HEAPSTER_PLUGIN",
	MagicCookieValue: "heapster",
}

var PluginMap = make(map[string]plugin.Plugin)

type GoPluginSink interface {
	Stop()
	Name() string
	Plugin(name string) (interface{}, error)
}

type goPluginSink struct {
	*plugin.Client
}

func (sink *goPluginSink) Stop() {
	sink.Kill()
}

func (sink *goPluginSink) Name() string {
	return "Go-Plugin Sink"
}

func (sink *goPluginSink) Plugin(name string) (interface{}, error) {
	c, err := sink.Client.Client()
	if err != nil {
		return nil, err
	}
	s, err := c.Dispense(name)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func NewGoPluginClient(uri *url.URL) (GoPluginSink, error) {
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url's query string: %s", err)
	}
	if len(opts["cmd"]) == 0 {
		return nil, fmt.Errorf("cmd parameter is mandatory %v", uri)
	}
	cmdArgs := opts["cmd"][0]
	var args []string
	if len(opts["cmd"]) > 1 {
		args = opts["cmd"][1:]
	}
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins:         PluginMap,
		Cmd:             exec.Command(cmdArgs, args...),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
	})
	return &goPluginSink{
		Client: client,
	}, nil
}
