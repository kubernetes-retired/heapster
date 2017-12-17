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
	"k8s.io/heapster/metrics/core"
	"sort"
	"strings"
)

type InfluxstatsdFormatter struct {
	delimReplacer *strings.Replacer
}

func (formatter *InfluxstatsdFormatter) Format(prefix string, name string, labels map[string]string, customizeLabel CustomizeLabel, metricValue core.MetricValue) (res string, err error) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s%s", formatter.delimReplacer.Replace(prefix), formatter.delimReplacer.Replace(name)))
	expandedLabels := formatter.expandUserLabels(labels)
	keys := make([]string, len(expandedLabels))
	for k := range expandedLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := expandedLabels[k]
		if v != "" {
			buffer.WriteString(fmt.Sprintf(",%s=%s", customizeLabel(formatter.delimReplacer.Replace(k)), formatter.delimReplacer.Replace(v)))
		}
	}
	buffer.WriteString(fmt.Sprintf(":%v|g", metricValue.GetValue()))

	return buffer.String(), nil
}

func (formatter *InfluxstatsdFormatter) expandUserLabels(labels map[string]string) map[string]string {
	res := make(map[string]string)
	var userLabelStr string
	for k, v := range labels {
		if k == core.LabelLabels.Key {
			userLabelStr = v
		} else {
			res[k] = v
		}
	}
	kvPairs := strings.Split(userLabelStr, ",")
	for _, kvPair := range kvPairs {
		kv := strings.Split(kvPair, ":")
		if len(kv) >= 2 {
			res[kv[0]] = kv[1]
		}
	}
	return res
}

func NewInfluxstatsdFormatter() Formatter {
	glog.V(2).Info("influxstatsd formatter is created")
	return &InfluxstatsdFormatter{
		delimReplacer: strings.NewReplacer(",", "_", ":", "_", "=", "_", "|", "_"),
	}
}
