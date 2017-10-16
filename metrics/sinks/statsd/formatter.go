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
	"fmt"
	"k8s.io/heapster/metrics/core"
	"strings"
	"unicode"
)

type Formatter interface {
	Format(prefix string, name string, labels map[string]string, customizeLabel CustomizeLabel, metricValue core.MetricValue) (res string, err error)
}

type LabelStyle func(string) string
type CustomizeLabel func(string) string

type LabelCustomizer struct {
	renameLabels map[string]string
	labelStyle   LabelStyle
}

func (customizer LabelCustomizer) Customize(label string) string {
	if val, ok := customizer.renameLabels[label]; ok {
		label = val
	}
	return (customizer.labelStyle(label))
}

func NewFormatter(protocolType string) (formatter Formatter, err error) {
	switch protocolType {
	case "etsystatsd":
		return NewEtsystatsdFormatter(), nil
	case "influxstatsd":
		return NewInfluxstatsdFormatter(), nil
	default:
		return nil, fmt.Errorf("Unknown statd formatter %s", protocolType)
	}
}

func DefaultLabelStyle(str string) string {
	return str
}

func SnakeToLowerCamel(str string) string {
	res := SnakeToUpperCamel(str)

	ustr := []rune(res)
	ustr[0] = unicode.ToLower(ustr[0])
	res = string(ustr)

	return res
}

func SnakeToUpperCamel(str string) string {
	res := ""
	words := strings.Split(str, "_")
	for _, word := range words {
		res += strings.Title(word)
	}
	return res
}
