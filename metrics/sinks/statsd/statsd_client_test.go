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

package statsd

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	validHost             = "127.0.0.1:48125"
	validNumMetricsPerMsg = 3
	bufferSize            = 1024
)

var msgs = [...]string{
	"test message 0",
	"test message 1",
	"test message 2",
	"test message 3",
	"test message 4",
	"test message 5",
	"test message 6",
	"test message 7",
}

func TestInvalidHostname(t *testing.T) {
	client, err := NewStatsdClient("badhostname:8125", validNumMetricsPerMsg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	err = client.open()
	assert.Error(t, err, "An error to lookup an invalid host was expected")
}

func TestInvalidPortNumber(t *testing.T) {
	client, err := NewStatsdClient("localhost", validNumMetricsPerMsg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	err = client.open()
	assert.Error(t, err, "Error expected - missing port number")

	client, err = NewStatsdClient("localhost:-8125", validNumMetricsPerMsg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	err = client.open()
	assert.Error(t, err, "Error expected - port number cannot be negative")
}

func TestInvalidNumMetricsPerMsg(t *testing.T) {
	_, err := NewStatsdClient(validHost, 0)
	assert.Error(t, err, "Error expected - number of metrics per message cannot be 0")

	_, err = NewStatsdClient(validHost, -1)
	assert.Error(t, err, "Error expected - number of metrics per message cannot be negative")
}

func TestClose(t *testing.T) {
	client, err := NewStatsdClient(validHost, validNumMetricsPerMsg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	err = client.close()
	assert.NoError(t, err, "Unexpected error - close() should be a NOOP if it is called before open()")

	err = client.open()
	assert.NoError(t, err)
	err = client.close()
	assert.NoError(t, err)
	err = client.close()
	assert.NoError(t, err, "Unexpected error - close() should be a NOOP if it is called after another close()")
}

func initClientServer(t *testing.T, messages []string, numMetricsPerMsg int) (client statsdClient, serverConn *net.UDPConn) {
	client, err := NewStatsdClient(validHost, numMetricsPerMsg)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.open()
	assert.NoError(t, err)

	addr, err := net.ResolveUDPAddr("udp", validHost)
	assert.NoError(t, err)
	assert.NotNil(t, addr)

	conn, err := net.ListenUDP("udp", addr)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	err = client.send(messages)
	assert.NoError(t, err)

	return client, conn
}

func TestSendOneMsg(t *testing.T) {

	numMetricsPerMsg := 10
	start := 0
	end := 5
	client, conn := initClientServer(t, msgs[start:end], numMetricsPerMsg)

	expectedMsg := strings.Join(msgs[start:end], "\n") + "\n"

	buf := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buf)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, string(buf[0:n]))

	err = client.close()
	assert.NoError(t, err)
	conn.Close()
}

func TestSendMultipleMsgsEqualBatches(t *testing.T) {

	buf := make([]byte, bufferSize)
	numMetricsPerMsg := 4
	start := 0
	end := 8
	client, conn := initClientServer(t, msgs[start:end], numMetricsPerMsg)

	expectedMsg := strings.Join(msgs[0:4], "\n") + "\n"
	n, _, err := conn.ReadFromUDP(buf)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, string(buf[0:n]))

	expectedMsg = strings.Join(msgs[4:8], "\n") + "\n"
	n, _, err = conn.ReadFromUDP(buf)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, string(buf[0:n]))

	err = client.close()
	assert.NoError(t, err)
	conn.Close()
}

func TestSendMultipleMsgsUnequalBatches(t *testing.T) {

	buf := make([]byte, bufferSize)
	numMetricsPerMsg := 3
	start := 0
	end := 8
	client, conn := initClientServer(t, msgs[start:end], numMetricsPerMsg)

	expectedMsg := strings.Join(msgs[0:3], "\n") + "\n"
	n, _, err := conn.ReadFromUDP(buf)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, string(buf[0:n]))

	expectedMsg = strings.Join(msgs[3:6], "\n") + "\n"
	n, _, err = conn.ReadFromUDP(buf)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, string(buf[0:n]))

	expectedMsg = strings.Join(msgs[6:8], "\n") + "\n"
	n, _, err = conn.ReadFromUDP(buf)
	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, string(buf[0:n]))

	err = client.close()
	assert.NoError(t, err)
	conn.Close()
}
