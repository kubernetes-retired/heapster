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

package push

import (
	"strings"
	"time"

	"k8s.io/heapster/metrics/core"
)

// MetricValue is a MetricValue that can also represent an accumulator to later be averaged
type MetricValue struct {
	// The sum value/normal value of the metric (possibly divided by StoredCount to get the final value)
	core.MetricValue

	// StoredCount is the divisor used when taking the average in the case of guage metrics
	StoredCount int
}

func (v *MetricValue) FinalValue() core.MetricValue {
	// no need to do anything fancy if this is not a gauge or only has one data point
	if v.MetricType != core.MetricGauge || v.StoredCount < 2 {
		return v.MetricValue
	}

	res := core.MetricValue{
		ValueType:  v.ValueType,
		MetricType: v.MetricType,
	}

	// take the average
	switch v.ValueType {
	case core.ValueInt64:
		res.IntValue = v.IntValue / int64(v.StoredCount)
	case core.ValueFloat:
		res.FloatValue = v.FloatValue / float32(v.StoredCount)
	}

	return res
}

// Append return the new value (in case of a non-Gauge), or returns this value with the relavant
// fields summed/incremented in case of a guage (for later averaging)
func (v MetricValue) Append(newVal MetricValue) MetricValue {
	if v.MetricType != core.MetricGauge {
		return newVal
	}

	if v.StoredCount == 0 {
		v.StoredCount = 2
	} else {
		v.StoredCount++
	}

	switch v.ValueType {
	case core.ValueInt64:
		v.IntValue += newVal.IntValue
	case core.ValueFloat:
		v.FloatValue += newVal.FloatValue
	}

	return v
}

// LabelPair is a key-value pair representing a label on a labeled metric
type LabelPair struct {
	Key   string
	Value string
}

// LabelPairs implements sort.Interface for LabelPair
type LabelPairs []LabelPair

func (s LabelPairs) Len() int           { return len(s) }
func (s LabelPairs) Less(i, j int) bool { return s[i].Key < s[j].Key }
func (s LabelPairs) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// A representation of core.LabeledMetric amenable to easy comparison
type LabeledMetric struct {
	// The Name of the metric
	Name string
	// Key-Value pairs representing the labels applied to the metric.
	// Should be sorted by key.
	Labels LabelPairs
	// The value of the metric
	MetricValue
}

// Append return the new value (in case of a non-Gauge), or returns this value with the relavant
// fields summed/incremented in case of a guage (for later averaging)
func (v LabeledMetric) Append(newVal LabeledMetric) LabeledMetric {
	if v.MetricType != core.MetricGauge {
		return newVal
	}

	v.MetricValue = v.MetricValue.Append(newVal.MetricValue)

	return v
}

func (v LabeledMetric) FinalValue() core.LabeledMetric {
	res := core.LabeledMetric{
		Name:        v.Name,
		Labels:      make(map[string]string, len(v.Labels)),
		MetricValue: v.MetricValue.FinalValue(),
	}

	for _, pair := range v.Labels {
		res.Labels[pair.Key] = pair.Value
	}

	return res
}

// LabeledMetricID is a comparable form the identifying information for a LabeledMetric
type LabeledMetricID struct {
	// The Name of a Metric
	Name string

	// An opaque, unique representation of the metric labels and values
	// (currently sorted, null-separated key-value pairs)
	JoinedLabels string
}

func (v *LabeledMetric) MakeID() LabeledMetricID {
	labelList := make([]string, len(v.Labels)*2)
	for i, lblPair := range v.Labels {
		labelList[i*2] = lblPair.Key
		labelList[i*2+1] = lblPair.Value
	}

	return LabeledMetricID{
		Name:         v.Name,
		JoinedLabels: strings.Join(labelList, "\000"),
	}
}

type MetricSet struct {
	// MetricValues are non-labeled metrics (mapping metric names to values)
	MetricValues map[string]MetricValue
	// Labels are the global lables associated with the key for this metric set
	Labels map[string]string
	// Labeled metrics are the labeled metrics for this key (mapping an unique
	// identifier to the labeled metric for use in merging)
	LabeledMetrics map[LabeledMetricID]LabeledMetric
}

// MergeInto extracts this push metric set into another push metric set, merging the two
func (s *MetricSet) MergeInto(targetSet *MetricSet, overwrite bool) {
	// add in normal labels
	if len(s.Labels) > 0 && targetSet.Labels == nil {
		targetSet.Labels = make(map[string]string, len(s.Labels))
	}

	for lblName, lblValue := range s.Labels {
		if !overwrite {
			if _, oldValPresent := targetSet.Labels[lblName]; oldValPresent {
				continue
			}
		}
		targetSet.Labels[lblName] = lblValue
	}

	// add in normal values
	if len(s.MetricValues) > 0 && targetSet.MetricValues == nil {
		targetSet.MetricValues = make(map[string]MetricValue)
	}

	for metricName, pushValue := range s.MetricValues {
		if !overwrite {
			if _, oldValPresent := targetSet.MetricValues[metricName]; oldValPresent {
				continue
			}
			targetSet.MetricValues[metricName] = pushValue
			continue
		}

		targetSet.MetricValues[metricName] = targetSet.MetricValues[metricName].Append(pushValue)
	}

	// add in the labeled metrics
	if len(s.LabeledMetrics) > 0 && targetSet.LabeledMetrics == nil {
		targetSet.LabeledMetrics = make(map[LabeledMetricID]LabeledMetric, len(targetSet.LabeledMetrics))
	}

	for lblID, labeledMetric := range s.LabeledMetrics {
		if !overwrite {
			if _, oldValPresent := targetSet.LabeledMetrics[lblID]; oldValPresent {
				continue
			}
			targetSet.LabeledMetrics[lblID] = labeledMetric
			continue
		}

		targetSet.LabeledMetrics[lblID] = targetSet.LabeledMetrics[lblID].Append(labeledMetric)
	}
}

func (s *MetricSet) MetricSet() *core.MetricSet {
	res := &core.MetricSet{
		MetricValues:   make(map[string]core.MetricValue, len(s.MetricValues)),
		Labels:         s.Labels,
		LabeledMetrics: make([]core.LabeledMetric, 0, len(s.LabeledMetrics)),
	}

	for metricName, pushValue := range s.MetricValues {
		res.MetricValues[metricName] = pushValue.FinalValue()
	}

	for _, pushLabeledMetric := range s.LabeledMetrics {
		res.LabeledMetrics = append(res.LabeledMetrics, pushLabeledMetric.FinalValue())
	}

	return res
}

// DataBatch is a core.DataBatch for push metrics
type DataBatch struct {
	Timestamp  time.Time
	MetricSets map[string]*MetricSet
}

func (b *DataBatch) DataBatch() *core.DataBatch {
	res := &core.DataBatch{
		Timestamp:  b.Timestamp,
		MetricSets: make(map[string]*core.MetricSet, len(b.MetricSets)),
	}

	for key, pushSet := range b.MetricSets {
		res.MetricSets[key] = pushSet.MetricSet()
	}

	return res
}

// MergeInto extracts this push batch's metric sets into another push batch's metric sets, merging the two
func (b *DataBatch) MergeInto(targetSets map[string]*MetricSet, overwrite bool) {
	for setKey, pushSet := range b.MetricSets {
		presentSet, wasPresent := targetSets[setKey]
		if !wasPresent {
			presentSet = &MetricSet{}
			targetSets[setKey] = presentSet
		}

		pushSet.MergeInto(presentSet, overwrite)
	}
}
