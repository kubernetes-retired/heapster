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

type EtsystatsdFormatter struct {
	delimReplacer *strings.Replacer
}

func (formatter *EtsystatsdFormatter) Format(prefix string, name string, labels map[string]string, customizeLabel CustomizeLabel, metricValue core.MetricValue) (res string, err error) {
	var metricName string
	var suffix string
	if resourceId, ok := labels[core.LabelResourceID.Key]; ok {
		metricName = fmt.Sprintf("%s.%s",
			formatter.delimReplacer.Replace(name),
			formatter.delimReplacer.Replace(resourceId))
	} else {
		metricName = formatter.delimReplacer.Replace(name)
	}
	if prefix != "" {
		prefix = formatter.delimReplacer.Replace(prefix) + "."
	}
	userLabelStr := labels[core.LabelLabels.Key]
	if len(userLabelStr) > 0 {
		suffix = fmt.Sprintf("%s.%s:%v|g",
			formatter.formatUserLabels(userLabelStr, customizeLabel),
			metricName,
			metricValue.GetValue(),
		)
	} else {
		suffix = fmt.Sprintf("%s:%v|g",
			metricName,
			metricValue.GetValue(),
		)
	}
	metricType, hasMetricType := labels[core.LabelMetricSetType.Key]
	if !hasMetricType {
		return "", fmt.Errorf("Missing LabelMetricSetType in labels, failed to format metrics")
	}
	switch metricType {
	case core.MetricSetTypePodContainer:
		return fmt.Sprintf("%snode.%s.namespace.%s.pod.%s.container.%s.%s",
			prefix,
			formatter.delimReplacer.Replace(labels[core.LabelHostname.Key]),
			formatter.delimReplacer.Replace(labels[core.LabelNamespaceName.Key]),
			formatter.delimReplacer.Replace(labels[core.LabelPodName.Key]),
			formatter.delimReplacer.Replace(labels[core.LabelContainerName.Key]),
			suffix,
		), nil
	case core.MetricSetTypeSystemContainer:
		return fmt.Sprintf("%snode.%s.sys-container.%s.%s",
			prefix,
			formatter.delimReplacer.Replace(labels[core.LabelHostname.Key]),
			formatter.delimReplacer.Replace(labels[core.LabelContainerName.Key]),
			suffix,
		), nil
	case core.MetricSetTypePod:
		return fmt.Sprintf("%snode.%s.namespace.%s.pod.%s.%s",
			prefix,
			formatter.delimReplacer.Replace(labels[core.LabelHostname.Key]),
			formatter.delimReplacer.Replace(labels[core.LabelNamespaceName.Key]),
			formatter.delimReplacer.Replace(labels[core.LabelPodName.Key]),
			suffix,
		), nil
	case core.MetricSetTypeNamespace:
		return fmt.Sprintf("%snamespace.%s.%s",
			prefix,
			formatter.delimReplacer.Replace(labels[core.LabelNamespaceName.Key]),
			suffix,
		), nil
	case core.MetricSetTypeNode:
		return fmt.Sprintf("%snode.%s.%s",
			prefix,
			formatter.delimReplacer.Replace(labels[core.LabelHostname.Key]),
			suffix,
		), nil
	case core.MetricSetTypeCluster:
		return fmt.Sprintf("%scluster.%s",
			prefix,
			suffix,
		), nil
	default:
		err = fmt.Errorf("Unknown metric set type %s", metricType)
	}
	return "", err
}

func (formatter *EtsystatsdFormatter) formatUserLabels(appLabel string, customizeLabel CustomizeLabel) string {
	labelMap := make(map[string]string)
	kvPairs := strings.Split(appLabel, ",")
	for _, kvPair := range kvPairs {
		kv := strings.Split(kvPair, ":")
		if len(kv) >= 2 {
			labelMap[kv[0]] = kv[1]
		}
	}
	keys := make([]string, len(labelMap))
	for k := range labelMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buffer bytes.Buffer
	for _, k := range keys {
		v := labelMap[k]
		if v != "" && buffer.Len() > 0 {
			buffer.WriteString(fmt.Sprintf(".%s.%s", customizeLabel(formatter.delimReplacer.Replace(k)), formatter.delimReplacer.Replace(v)))
		} else if v != "" {
			buffer.WriteString(fmt.Sprintf("%s.%s", customizeLabel(formatter.delimReplacer.Replace(k)), formatter.delimReplacer.Replace(v)))
		}
	}
	return buffer.String()
}

func NewEtsystatsdFormatter() Formatter {
	glog.V(2).Info("etsystatsd formatter is created")
	return &EtsystatsdFormatter{
		delimReplacer: strings.NewReplacer(".", "_", "=", "_", "|", "_"),
	}
}
