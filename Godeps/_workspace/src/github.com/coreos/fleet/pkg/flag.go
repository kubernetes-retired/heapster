// Copyright 2015 CoreOS, Inc.
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

package pkg

import (
	"fmt"
	"strings"
)

type StringSlice []string

func (f *StringSlice) Set(value string) error {
	var s StringSlice
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimLeft(item, " [\"")
		item = strings.TrimRight(item, " \"]")
		s = append(s, item)
	}

	*f = s

	return nil
}

func (f *StringSlice) String() string {
	return fmt.Sprintf("%v", *f)
}

func (f *StringSlice) Value() []string {
	return *f
}

func (f *StringSlice) Get() interface{} {
	return *f
}
