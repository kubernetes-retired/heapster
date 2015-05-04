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

package bosun

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"bosun.org/metadata"
	"bosun.org/opentsdb"

	"github.com/GoogleCloudPlatform/heapster/extpoints"
	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api"
	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/golang/glog"
)

const (
	defaultBosunUrl = "http://monitoring-bosun:80"
	defaultTags     = "name=heapster"
	defaultRoot     = "kubernetes.io"
)

func init() {
	extpoints.SinkFactories.Register(CreateBosun, "bosun")
}

func CreateBosun(uri string, options map[string][]string) ([]sink_api.ExternalSink, error) {
	if len(uri) == 0 {
		uri = defaultBosunUrl
	}
	parsedUrl, err := url.Parse(os.ExpandEnv(uri))
	if err != nil {
		return nil, err
	}
	metaUrl, err := parsedUrl.Parse("/api/metadata/put")
	if err != nil {
		return nil, err
	}
	metricsUrl, err := parsedUrl.Parse("/api/put")
	if err != nil {
		return nil, err
	}
	client := &bosunClientImpl{
		metricsUrl: metricsUrl,
		metaUrl:    metaUrl,
		client: &http.Client{
			Timeout: time.Minute,
		},
	}
	glog.Infof("Created Bosun Sink - Metrics URL: %q, Metadata URL: %q", metricsUrl.String(), metaUrl.String())
	return []sink_api.ExternalSink{&bosunSink{client}}, nil
}

type bosunSink struct {
	client bosunClient
}

func (bs *bosunSink) Register(metrics []sink_api.MetricDescriptor) error {
	ms := []metadata.Metasend{}
	for _, metric := range metrics {
		rate := ""
		switch metric.Type {
		case sink_api.MetricCumulative:
			rate = "counter"
			break
		case sink_api.MetricGauge:
			rate = "gauge"
			break
		}

		ms = append(ms, metadata.Metasend{
			Metric: bs.fullMetricName(metric.Name),
			Name:   "rate",
			Value:  rate,
		})
		ms = append(ms, metadata.Metasend{
			Metric: bs.fullMetricName(metric.Name),
			Name:   "unit",
			Value:  metric.Units.String(),
		})
		ms = append(ms, metadata.Metasend{
			Metric: bs.fullMetricName(metric.Name),
			Name:   "desc",
			Value:  metric.Description,
		})
	}
	glog.Infof("Registering Metrics with Bosun: %v", ms)
	return bs.client.sendMetadata(ms)
}

func (bs *bosunSink) fullMetricName(name string) string {
	return strings.Replace(path.Join(defaultRoot, name), "/", "_", -1)
}

func (bs *bosunSink) getpoint(ts sink_api.Timeseries) (*opentsdb.DataPoint, error) {
	tags := opentsdb.TagSet{}
	for key, val := range ts.Point.Labels {
		if key == sink_api.LabelLabels {
			// TODO: Handle support for user specified labels.
			continue
		}
		tag, err := opentsdb.ParseTags(fmt.Sprintf("%s=%s", key, val))
		if err != nil {
			glog.Errorf("failed to parse label [%v](%v) - %v", key, val, err)
			return nil, err
		}
		tags.Merge(tag)
	}
	return &opentsdb.DataPoint{
		Metric:    bs.fullMetricName(ts.Point.Name),
		Timestamp: ts.Point.End.UnixNano() / int64(time.Millisecond),
		Value:     ts.Point.Value,
		Tags:      tags,
	}, nil
}

func (bs *bosunSink) StoreTimeseries(timeseries []sink_api.Timeseries) error {
	dps := []json.RawMessage{}
	for _, ts := range timeseries {
		point, err := bs.getpoint(ts)
		if err != nil {
			return err
		}
		m, err := json.Marshal(point)
		if err != nil {
			glog.V(2).Infof("failed to marshal datapoint %v - %v", point, err)
			return err
		}
		dps = append(dps, m)
	}
	return bs.client.sendBatch(dps)
}

func (bs *bosunSink) StoreEvents(events []kube_api.Event) error {
	return nil
}

func (bs *bosunSink) DebugInfo() string {
	desc := "Sink Type: Bosun\n"
	desc += "\n"
	return desc
}
