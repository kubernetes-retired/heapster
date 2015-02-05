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

package gcm

import (
	"fmt"

	"github.com/GoogleCloudPlatform/heapster/sources"
)

type GcmSink struct {
	// The driver for GCM interactions.
	driver *gcmDriver
}

func NewSink() (*GcmSink, error) {
	driver, err := NewDriver()
	if err != nil {
		return nil, err
	}

	return &GcmSink{
		driver: driver,
	}, nil
}

func (self *GcmSink) StoreData(data interface{}) error {
	data, ok := data.(sources.ContainerData)
	if !ok {
		return fmt.Errorf("requesting unrecognized type to be stored in GCM")
	}

	// TODO(vmarmol): Implement.

	return nil
}

func (self *GcmSink) GetConfig() string {
	desc := "Sink type: GCM\n"
	desc += "\n"
	return desc
}
