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

package gce

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	waitForTokenInterval = 1 * time.Second
	waitForTokenTimeout  = 30 * time.Second
)

func GetGCEToken() (oauth2.TokenSource, error) {
	token := google.ComputeTokenSource("")
	for start := time.Now(); time.Since(start) < waitForTokenTimeout; time.Sleep(waitForTokenInterval) {
		if _, err := token.Token(); err != nil {
			glog.Errorf("Waiting for GCE token error: %v", err)
		} else {
			return token, nil
		}
	}
	return nil, fmt.Errorf("timeout while waiting for GCE token")
}
