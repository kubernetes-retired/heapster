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

package schema

import (
	"time"

	"github.com/GoogleCloudPlatform/heapster/store"
)

// maxTimestamp returns its largest time.Time argument
func maxTimestamp(first time.Time, second time.Time) time.Time {
	if first.After(second) {
		return first
	}
	return second
}

// newInfoType is an InfoType Constructor, which returns a new InfoType.
// Initial fields for the new InfoType can be provided as arguments.
// A nil argument results in a newly-allocated map for that field.
func newInfoType(metrics map[string]*store.TimeStore, labels map[string]string) InfoType {
	if metrics == nil {
		metrics = make(map[string]*store.TimeStore)
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	return InfoType{
		Metrics: metrics,
		Labels:  labels,
	}
}
