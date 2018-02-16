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

package statsd

import (
	"fmt"
	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
	"sort"
	"strings"
)

type DogstatsdFormatter struct {
	delimReplacer *strings.Replacer
}

func (formatter *DogstatsdFormatter) Format(prefix string, name string, labels map[string]string, customizeLabel CustomizeLabel, metricValue core.MetricValue) (res string, err error) {
	expandedLabels := formatter.expandUserLabels(labels)
	keys := make([]string, len(expandedLabels))
	finalizedLabels := make([]string, len(expandedLabels))
	for k := range expandedLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	idx := 0
	for _, k := range keys {
		v := expandedLabels[k]
		if v != "" {
			finalizedLabels[idx] = fmt.Sprintf("%s:%s", customizeLabel(formatter.delimReplacer.Replace(k)), formatter.delimReplacer.Replace(v))
			idx++
		}
	}

	res = fmt.Sprintf("%s:%v|g|#%s",
		fmt.Sprintf("%s%s", formatter.delimReplacer.Replace(prefix), formatter.delimReplacer.Replace(name)), metricValue.GetValue(),
		strings.Join(finalizedLabels, ","))

	return res, nil
}

func (formatter *DogstatsdFormatter) expandUserLabels(labels map[string]string) map[string]string {
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

func NewDogstatsdFormatter() Formatter {
	glog.V(2).Info("Dogstatsd formatter is created")
	return &DogstatsdFormatter{
		delimReplacer: strings.NewReplacer(",", "_", ":", "_", "=", "_", "|", "_"),
	}
}
