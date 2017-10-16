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
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"net"
)

type statsdClient interface {
	open() error
	close() error
	send(messages []string) error
}

type statsdClientImpl struct {
	host             string
	numMetricsPerMsg int
	conn             net.Conn
}

func (client *statsdClientImpl) open() error {
	var err error
	client.conn, err = net.Dial("udp", client.host)
	if err != nil {
		glog.Errorf("Failed to open statsd client connection : %v", err)
	} else {
		glog.V(2).Infof("statsd client connection opened : %+v", client.conn)
	}
	return err
}

func (client *statsdClientImpl) close() error {
	if client.conn == nil {
		glog.Info("statsd client connection already closed")
		return nil
	}
	err := client.conn.Close()
	client.conn = nil
	glog.V(2).Infof("statsd client connection closed")
	return err
}

func (client *statsdClientImpl) send(messages []string) error {
	if client.conn == nil {
		err := client.open()
		if err != nil {
			return fmt.Errorf("send() failed - %v", err)
		}
	}
	var numMetrics = 0
	var err, tmpErr error
	buf := bytes.NewBufferString("")
	for _, msg := range messages {
		buf.WriteString(fmt.Sprintf("%s\n", msg))
		numMetrics++
		if numMetrics >= client.numMetricsPerMsg {
			_, tmpErr = client.conn.Write(buf.Bytes())
			if tmpErr != nil {
				err = tmpErr
			}
			buf.Reset()
			numMetrics = 0
		}
	}
	if buf.Len() > 0 {
		_, tmpErr = client.conn.Write(buf.Bytes())
		if tmpErr != nil {
			err = tmpErr
		}
	}
	return err
}

func NewStatsdClient(host string, numMetricsPerMsg int) (client statsdClient, err error) {
	if numMetricsPerMsg <= 0 {
		return nil, fmt.Errorf("numMetricsPerMsg should be a positive integer : %d", numMetricsPerMsg)
	}
	glog.V(2).Infof("statsd client created")
	return &statsdClientImpl{host: host, numMetricsPerMsg: numMetricsPerMsg}, nil
}
