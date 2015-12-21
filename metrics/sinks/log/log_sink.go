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

package logsink

import (
	"bytes"
	"fmt"

	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
)

type LogSink struct {
}

func (this *LogSink) Name() string {
	return "LogSink"
}

func (this *LogSink) Stop() {
	// Do nothing.
}

func batchToString(batch *core.DataBatch) string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("DataBatch     Timestamp: %s\n\n", batch.Timestamp))
	for key, ms := range batch.MetricSets {
		buffer.WriteString(fmt.Sprintf("MetricSet: %s\n", key))
		padding := "   "
		buffer.WriteString(fmt.Sprintf("%sLabels:\n", padding))
		for labelName, labelValue := range ms.Labels {
			buffer.WriteString(fmt.Sprintf("%s%s%s = %s\n", padding, padding, labelName, labelValue))
		}
		buffer.WriteString(fmt.Sprintf("%sMetrics:\n", padding))
		for metricName, metricValue := range ms.MetricValues {
			if core.ValueInt64 == metricValue.ValueType {
				buffer.WriteString(fmt.Sprintf("%s%s%s = %d\n", padding, padding, metricName, metricValue.IntValue))
			} else if core.ValueFloat == metricValue.ValueType {
				buffer.WriteString(fmt.Sprintf("%s%s%s = %f\n", padding, padding, metricName, metricValue.FloatValue))
			} else {
				buffer.WriteString(fmt.Sprintf("%s%s%s = ?\n", padding, padding, metricName))
			}
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func (this *LogSink) ExportData(batch *core.DataBatch) {
	glog.Info(batchToString(batch))
}

func NewLogSink() *LogSink {
	return &LogSink{}
}
