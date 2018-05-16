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
	"os"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
)

func TestNewGoPluginClient(t *testing.T) {
	uri, err := url.Parse(fmt.Sprintf("plugin:?cmd=%s&cmd=-test.run=TestHelperProcess&cmd=--&cmd=mock", os.Args[0]))
	assert.NoError(t, err)
	client, err := NewGoPluginClient(uri)
	assert.NoError(t, err)
	defer client.Stop()
}

func TestHelperProcess(*testing.T) {
	if os.Getenv(Handshake.MagicCookieKey) == "" {
		return
	}

	defer os.Exit(0)
	args := os.Args
	cmd, args := args[0], args[1:]
	switch cmd {
	case "mock":
		fmt.Printf("%d|%d|tcp|:1234\n", plugin.CoreProtocolVersion, Handshake.ProtocolVersion)
		<-make(chan int)
	}
}
