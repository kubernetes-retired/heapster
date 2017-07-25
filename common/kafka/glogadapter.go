// Copyright 2014 Google Inc. All Rights Reserved.
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
package kafka

// GologAdapterLogger is an adapter between golog and the Logger
// interface defined in kafka api.
//
// * https://github.com/optiopay/kafka/blob/master/log.go

import (
	"github.com/golang/glog"
)

type GologAdapterLogger struct {
}

func (GologAdapterLogger) Debug(msg string, args ...interface{}) {
	glog.V(6).Infof(msg, args)
}

func (GologAdapterLogger) Info(msg string, args ...interface{}) {
	glog.Infof(msg, args)
}

func (GologAdapterLogger) Warn(msg string, args ...interface{}) {
	glog.Warningf(msg, args)
}

func (GologAdapterLogger) Error(msg string, args ...interface{}) {
	glog.Errorf(msg, args)
}
