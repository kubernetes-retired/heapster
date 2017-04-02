// Copyright 2017 Google Inc. All Rights Reserved.
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

package vpa

import (
	"github.com/golang/glog"
	"k8s.io/heapster/common/vpa"
	"k8s.io/heapster/metrics/core"
	"net/url"
	"sync"
)

type recommenderSink struct {
	recommenderClient vpa.JSONClient
	sync.RWMutex
}

func CreateRecommenderSink(uri *url.URL) (core.DataSink, error) {
	jsonClient := vpa.CreateRecommenderClient(uri.String())
	recommenderSink := recommenderSink{recommenderClient: jsonClient}

	return &recommenderSink, nil
}

func (sink *recommenderSink) ExportData(dataBatch *core.DataBatch) {
	sink.Lock()
	defer sink.Unlock()

	snapshotSet := utilizationSnapshotSet{}
	for _, metricSet := range dataBatch.MetricSets {
		if !isContainerMetricSet(metricSet) {
			continue //skip for non-container metrics
		}
		utilizationSnapshot, err := newContainerUtilizationSnapshot(metricSet)
		if err == nil {
			snapshotSet.addContainer(utilizationSnapshot)
		} else {
			glog.Warningf("Unable to create utilization snapshot, due to error: %s", err.Error())
		}
	}

	sink.sendAllSnapshots(&snapshotSet)
}

func (sink *recommenderSink) sendAllSnapshots(set *utilizationSnapshotSet) {
	if len(set.Containers) > 0 {
		_, err := sink.recommenderClient.SendJSON(set)
		if err != nil {
			glog.Errorf("Unable to send utilization snapshot to VPA recommender, due to error: %s", err)
		}
	}
}

func (sink *recommenderSink) Name() string {
	return "VPA Recommender Sink"
}

func (sink *recommenderSink) Stop() {
	// nothing needs to be done.
}

func isContainerMetricSet(metricSet *core.MetricSet) bool {
	return metricSet.Labels[core.LabelContainerName.Key] != "" && metricSet.Labels[core.LabelContainerBaseImage.Key] != ""
}
