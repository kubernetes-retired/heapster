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

package util

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/heapster/metrics/core"
)

// InternalUserHeader is the header set by the auth system to communicate the user extracted during auth
const UserAttributeName = "Heapster-User-Info"

// Concatenates a map of labels into a comma-separated key=value pairs.
func LabelsToString(labels map[string]string, separator string) string {
	output := make([]string, 0, len(labels))
	for key, value := range labels {
		output = append(output, fmt.Sprintf("%s:%s", key, value))
	}

	// Sort to produce a stable output.
	sort.Strings(output)
	return strings.Join(output, separator)
}

func CopyLabels(labels map[string]string) map[string]string {
	c := make(map[string]string, len(labels))
	for key, val := range labels {
		c[key] = val
	}
	return c
}

func GetLatest(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// MergeValuesFunc is a function for merging two maps of metric values.
type MergeValuesFunc func(destValues map[string]core.MetricValue, newValues map[string]core.MetricValue, overwrite bool)

// MergeMetricSetCustom works just like MergeMetricSet, except that a custom method for merging the actual metric
// values is accepted.
func MergeMetricSetCustom(destSet *core.MetricSet, newSet *core.MetricSet, overwrite bool, mergeValues MergeValuesFunc) {
	// copy over times if they're not already set
	if destSet.CreateTime.IsZero() || overwrite {
		destSet.CreateTime = newSet.CreateTime
	}

	if destSet.ScrapeTime.IsZero() || overwrite {
		destSet.ScrapeTime = newSet.ScrapeTime
	}

	// Merge labels
	for lblName, newLblVal := range newSet.Labels {
		if !overwrite {
			if _, oldValPresent := destSet.Labels[lblName]; oldValPresent {
				continue
			}
		}

		destSet.Labels[lblName] = newLblVal
	}

	// Merge metric values
	mergeValues(destSet.MetricValues, newSet.MetricValues, overwrite)

	// Append labled metric values

	// TODO: we should probably try to make sure there aren't any labeled metrics with
	// the same name and label set, instead of just appending here
	destSet.LabeledMetrics = append(destSet.LabeledMetrics, newSet.LabeledMetrics...)
}

// MergeMetricSet merges a new metric set into an existing (destination) metric set.
// If overwrite is true, if there are conflicts between label and metric values,
// the new set will overwrite the existing set.  Otherwise, conflicting values from
// the new set will be ignored.  Overwrite also causes times from the new set batch
// to replace those from the old one unconditionally (otherwise, replacement happens
// with missing values only).
func MergeMetricSet(destSet *core.MetricSet, newSet *core.MetricSet, overwrite bool) {

	MergeMetricSetCustom(destSet, newSet, overwrite, func(destValues map[string]core.MetricValue, newValues map[string]core.MetricValue, _ bool) {
		for valKey, newVal := range newValues {
			if !overwrite {
				if _, oldValPresent := destValues[valKey]; oldValPresent {
					continue
				}
			}

			destValues[valKey] = newVal
		}
	})
}
