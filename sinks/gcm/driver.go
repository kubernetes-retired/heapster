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

package gcm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/gcloud-golang/compute/metadata"
	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api"
	"github.com/golang/glog"
)

type gcmSink struct {
	// Token to use for authentication.
	token string

	// When the token expires.
	tokenExpiration time.Time

	// TODO(vmarmol): Make this configurable and not only detected.
	// GCE project.
	project string

	// TODO(vmarmol): Also store labels?
	// Map of metrics we currently export.
	exportedMetrics map[string]sink_api.MetricDescriptor
}

func (self *gcmSink) refreshToken() error {
	if time.Now().After(self.tokenExpiration) {
		token, err := getToken()
		if err != nil {
			return nil
		}

		// Expire the token a bit early.
		const earlyRefreshSeconds = 60
		if token.ExpiresIn > earlyRefreshSeconds {
			token.ExpiresIn -= earlyRefreshSeconds
		}
		self.token = token.AccessToken
		self.tokenExpiration = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	return nil
}

// GCM request structures for a MetricDescriptor.
type typeDescriptor struct {
	MetricType string `json:"metricType,omitempty"`
	ValueType  string `json:"valueType,omitempty"`
}

type metricDescriptor struct {
	Name           string                     `json:"name,omitempty"`
	Project        string                     `json:"project,omitempty"`
	Description    string                     `json:"description,omitempty"`
	Labels         []sink_api.LabelDescriptor `json:"labels,omitempty"`
	TypeDescriptor typeDescriptor             `json:"typeDescriptor,omitempty"`
}

const maxNumLabels = 10

// Adds the specified metrics or updates them if they already exist.
func (self *gcmSink) Register(metrics []sink_api.MetricDescriptor) error {
	for _, metric := range metrics {
		// Enforce the most labels that GCM allows.
		if len(metric.Labels) > maxNumLabels {
			return fmt.Errorf("metrics cannot have more than %d labels and %q has %d", maxNumLabels, metric.Name, len(metric.Labels))
		}

		// Ensure all labels are in the correct format.
		for i := range metric.Labels {
			metric.Labels[i].Key = fullLabelName(metric.Labels[i].Key)
		}

		request := metricDescriptor{
			Name:        fullMetricName(metric.Name),
			Project:     self.project,
			Description: metric.Description,
			Labels:      metric.Labels,
			TypeDescriptor: typeDescriptor{
				MetricType: metric.Type.String(),
				ValueType:  metric.ValueType.String(),
			},
		}

		err := sendRequest(fmt.Sprintf("https://www.googleapis.com/cloudmonitoring/v2beta2/projects/%s/metricDescriptors", self.project), self.token, request)
		glog.Infof("[GCM] Adding metric %q: %v", metric.Name, err)
		if err != nil {
			return err
		}

		// Add metric to exportedMetrics.
		self.exportedMetrics[metric.Name] = metric
	}

	return nil
}

// GCM request structures for writing time-series data.
type timeseriesDescriptor struct {
	Project string            `json:"project,omitempty"`
	Metric  string            `json:"metric,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type point struct {
	Start      time.Time `json:"start,omitempty"`
	End        time.Time `json:"end,omitempty"`
	Int64Value int64     `json:"int64Value"`
}

type timeseries struct {
	TimeseriesDescriptor timeseriesDescriptor `json:"timeseriesDesc,omitempty"`
	Point                point                `json:"point,omitempty"`
}

type metricWriteRequest struct {
	Timeseries []timeseries `json:"timeseries,omitempty"`
}

// The largest number of timeseries we can write to per request.
const maxTimeseriesPerRequest = 200

// Pushes the specified metric values in input. The metrics must already exist.
func (self *gcmSink) StoreTimeseries(inputTimeseriesList []sink_api.Timeseries) error {
	// Ensure the metrics exist.
	for _, inputTimeseries := range inputTimeseriesList {
		// TODO: Remove this check if possible.
		if _, ok := self.exportedMetrics[inputTimeseries.SeriesName]; !ok {
			return fmt.Errorf("unable to push unknown metric %q", inputTimeseries.SeriesName)
		}
	}

	// Build a map of metrics by name.
	metrics := make(map[string][]timeseries)
	for _, inputTimeseries := range inputTimeseriesList {
		// Ideally we should put multiple points in a single timeseries, but since GCM requires all points in a time series
		// to share a single set of labels, we can't do this.
		for _, inputPoint := range inputTimeseries.Points {
			// Use full label names.
			labels := make(map[string]string, len(inputPoint.Labels))
			for key, value := range inputPoint.Labels {
				labels[fullLabelName(key)] = value
			}

			// TODO(vmarmol): Validation and cleanup of data.
			// TODO(vmarmol): Handle non-int64 data types. There is an issue with using omitempty since 0 is a valid value for us.
			if _, ok := inputPoint.Value.(int64); !ok {
				return fmt.Errorf("non-int64 data not implemented. Seen for metric %q", inputTimeseries.SeriesName)
			}
			metrics[inputTimeseries.SeriesName] = append(metrics[inputTimeseries.SeriesName], timeseries{
				TimeseriesDescriptor: timeseriesDescriptor{
					Metric: fullMetricName(inputTimeseries.SeriesName),
					Labels: labels,
				},
				Point: point{
					Start:      inputPoint.Start,
					End:        inputPoint.End,
					Int64Value: inputPoint.Value.(int64),
				},
			})
		}
	}

	// Only send one metric of each type per request.
	var lastErr error
	for len(metrics) != 0 {
		var request metricWriteRequest
		for name, values := range metrics {
			// Remove metrics with no more values.
			if len(values) == 0 {
				delete(metrics, name)
				continue
			}

			m := values[0]
			metrics[name] = values[1:]
			request.Timeseries = append(request.Timeseries, m)
		}

		err := self.pushMetrics(&request)
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (self *gcmSink) pushMetrics(request *metricWriteRequest) error {
	if len(request.Timeseries) == 0 {
		return nil
	}
	// TODO(vmarmol): Split requests in this case.
	if len(request.Timeseries) > maxTimeseriesPerRequest {
		return fmt.Errorf("unable to write more than %d metrics at once and %d were provided", maxTimeseriesPerRequest, len(request.Timeseries))
	}

	// Refresh token.
	err := self.refreshToken()
	if err != nil {
		return err
	}

	const requestAttempts = 3
	for i := 0; i < requestAttempts; i++ {
		err = sendRequest(fmt.Sprintf("https://www.googleapis.com/cloudmonitoring/v2beta2/projects/%s/timeseries:write", self.project), self.token, request)
		if err != nil {
			glog.Warningf("[GCM] Push attempt %d failed: %v", i, err)
		} else {
			break
		}
	}
	if err != nil {
		prettyRequest, _ := json.MarshalIndent(request, "", "  ")
		glog.Warningf("[GCM] Pushing %d metrics \n%s\n failed: %v", len(request.Timeseries), string(prettyRequest), err)
	} else {
		glog.V(2).Infof("[GCM] Pushing %d metrics: SUCCESS", len(request.Timeseries))
	}
	return err
}

// Domain for the metrics.
const metricDomain = "kubernetes.io"

func fullLabelName(name string) string {
	if !strings.Contains(name, "custom.cloudmonitoring.googleapis.com/") {
		return fmt.Sprintf("custom.cloudmonitoring.googleapis.com/%s/label/%s", metricDomain, name)
	}
	return name
}

func fullMetricName(name string) string {
	if !strings.Contains(name, "custom.cloudmonitoring.googleapis.com/") {
		return fmt.Sprintf("custom.cloudmonitoring.googleapis.com/%s/%s", metricDomain, name)
	}
	return name
}

func sendRequest(url string, token string, request interface{}) error {
	rawRequest, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(rawRequest))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("request to %q failed with status %q and response: %q", url, resp.Status, string(out))
	}

	return nil
}

func (self *gcmSink) DebugInfo() string {
	return "Sink Type: GCM"
}

// Returns a thread-compatible implementation of GCM interactions.
func NewSink() (sink_api.ExternalSink, error) {
	// TODO: Retry OnGCE call for ~15 seconds before declaring failure.
	time.Sleep(3 * time.Second)
	// Only support GCE for now.
	if !metadata.OnGCE() {
		return nil, fmt.Errorf("the GCM sink is currently only supported on GCE")
	}

	// Detect project.
	project, err := metadata.ProjectID()
	if err != nil {
		return nil, err
	}

	// Check required service accounts
	err = checkServiceAccounts()
	if err != nil {
		return nil, err
	}

	impl := &gcmSink{
		project:         project,
		exportedMetrics: make(map[string]sink_api.MetricDescriptor),
	}

	// Get an initial token.
	err = impl.refreshToken()
	if err != nil {
		return nil, err
	}

	return impl, nil
}
