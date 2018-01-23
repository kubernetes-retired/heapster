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

package fluent

import (
        "github.com/golang/glog"
        fluent "github.com/lestrrat/go-fluent-client"
        "k8s.io/heapster/metrics/core"
        "net/url"
        "strconv"
        "sync"
)

type fluentSink struct {
        fluent.Client
        tag string
        sync.RWMutex
        fluent.Option
}

type FluentSinkPoint struct {
        MetricsName      string
        MetricsValue     interface{}
        MetricsTags      map[string]string
}

type fluentConfig struct {
        Host     string
        Tag      string
        Buffered bool
}

func (this *fluentSink) Stop() {
        // Do nothing.
}

func (sink *fluentSink) Name() string {
        return "Fluent Sink"
}

func (sink *fluentSink) ExportData(dataBatch *core.DataBatch) {
        sink.Lock()
        defer sink.Unlock()

        for _, metricSet := range dataBatch.MetricSets {
                for metricName, metricValue := range metricSet.MetricValues {
                        point := FluentSinkPoint{
                                MetricsName: metricName,
                                MetricsTags: metricSet.Labels,
                                MetricsValue: map[string]interface{}{
                                        "value": metricValue.GetValue(),
                                },
                        }

                        if err := sink.Post(sink.tag, point, fluent.WithTimestamp(dataBatch.Timestamp.UTC())); err != nil {
                                glog.Info("failed to post: %s", err)
                                return
                        }
                }
                for _, metric := range metricSet.LabeledMetrics {
                        labels := make(map[string]string)
                        for k, v := range metricSet.Labels {
                                labels[k] = v
                        }
                        for k, v := range metric.Labels {
                                labels[k] = v
                        }
                        point := FluentSinkPoint{
                                MetricsName: metric.Name,
                                MetricsTags: labels,
                                MetricsValue: map[string]interface{}{
                                        "value": metric.GetValue(),
                                },
                        }

                        if err := sink.Post(sink.tag, point, fluent.WithTimestamp(dataBatch.Timestamp.UTC())); err != nil {
                                glog.Info("failed to post: %s", err)
                                return
                        }
                }
        }
}

func NewFluentSink(uri *url.URL) (core.DataSink, error) {
        config := fluentConfig{
                Host:     "localhost:24224",
                Tag:      "fluent",
                Buffered: false,
        }

        config.Host = uri.Host
        options := uri.Query()

        // check options
        if len(options["tag"]) > 0 {
                config.Tag = options["tag"][0]
        }
        if len(options["bufferd"]) > 0 {
                b, err := strconv.ParseBool(options["bufferd"][0])
                if err == nil {
                        config.Buffered = b
                } else {
                        glog.Info("failed to convert %s to boolean", options["bufferd"][0], err)
                        return nil, err
                }
        }

        client, err := fluent.New(fluent.WithAddress(config.Host), fluent.WithBuffered(config.Buffered))
        if err != nil {
                // fluent.New may return an error if invalid values were
                // passed to the constructor
                glog.Info("failed to create client: %s", err)
                return nil, err
        }

        glog.Info("created fluent sink with options: host:%s tag:%s bufferd:%s", config.Host, config.Tag, config.Buffered)

        return &fluentSink{
                Client: client,
                tag:    config.Tag,
        }, nil
}
