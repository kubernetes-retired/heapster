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
	"k8s.io/heapster/metrics/core"
	"sort"
	"strings"
)

// LabelCopier maps kubernetes objects' labels to metrics
type LabelCopier struct {
	// labelSeparator contains separator used to join labels into "labels" label
	labelSeparator string
	// storedLabels maps source label names to their destination names used in metrics
	storedLabels map[string]string
	// ignoredLabels contains labels to be skipped during concatenation
	ignoredLabels map[string]string
}

// Copy copies the given set of pod labels into a set of metric labels, using the following logic:
// - all labels, unless found in ignoredLabels, are concatenated into a Separator-separated key:value pairs and stored under core.LabelLabels.Key
// - labels found in storedLabels are additionally stored under key provided
func (this *LabelCopier) Copy(in map[string]string, out map[string]string) {
	labels := make([]string, 0, len(in))

	for key, value := range in {
		if mappedKey, exists := this.storedLabels[key]; exists {
			out[mappedKey] = value
		}

		if _, exists := this.ignoredLabels[key]; !exists {
			labels = append(labels, fmt.Sprintf("%s:%s", key, value))
		}
	}

	sort.Strings(labels)
	out[core.LabelLabels.Key] = strings.Join(labels, this.labelSeparator)
}

// makeStoredLabels converts labels into a map for quicker retrieval.
// Incoming labels, if desired, may contain mappings in format "newName=oldName"
func makeStoredLabels(labels []string) map[string]string {
	storedLabels := make(map[string]string)
	for _, s := range labels {
		split := strings.SplitN(s, "=", 2)
		if len(split) == 1 {
			storedLabels[split[0]] = split[0]
		} else {
			storedLabels[split[1]] = split[0]
		}
	}
	return storedLabels
}

// makeIgnoredLabels converts label slice into a map for later use.
func makeIgnoredLabels(labels []string) map[string]string {
	ignoredLabels := make(map[string]string)
	for _, s := range labels {
		ignoredLabels[s] = ""
	}
	return ignoredLabels
}

// NewLabelCopier creates a new instance of LabelCopier type
func NewLabelCopier(separator string, storedLabels, ignoredLabels []string) (*LabelCopier, error) {
	return &LabelCopier{
		labelSeparator: separator,
		storedLabels:   makeStoredLabels(storedLabels),
		ignoredLabels:  makeIgnoredLabels(ignoredLabels),
	}, nil
}
