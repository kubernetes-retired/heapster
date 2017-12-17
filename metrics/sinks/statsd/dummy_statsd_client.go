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
	"github.com/golang/glog"
)

type dummyStatsdClientImpl struct {
	messages []string
}

func (client *dummyStatsdClientImpl) open() error {
	glog.V(2).Infof("dummy statsd client open() called, doing nothing")
	return nil
}

func (client *dummyStatsdClientImpl) close() error {
	glog.V(2).Infof("dummy statsd client close() called, doing nothing")
	return nil
}

func (client *dummyStatsdClientImpl) send(messages []string) error {
	client.messages = messages
	return nil
}
