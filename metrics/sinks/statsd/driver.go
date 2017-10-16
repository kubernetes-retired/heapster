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
	"fmt"
	"github.com/golang/glog"
	"k8s.io/heapster/metrics/core"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const (
	defaultHost             = "localhost:8125"
	defaultNumMetricsPerMsg = 5
	defaultProtocolType     = "etsystatsd"
)

type statsdSink struct {
	config    statsdConfig
	formatter Formatter
	client    statsdClient
	sync.RWMutex
}

type statsdConfig struct {
	host             string
	prefix           string
	numMetricsPerMsg int
	protocolType     string
	renameLabels     map[string]string
	allowedLabels    map[string]string
	customizeLabel   CustomizeLabel
}

func getConfig(uri *url.URL) (cfg statsdConfig, err error) {
	config := statsdConfig{
		host:             defaultHost,
		prefix:           "",
		numMetricsPerMsg: defaultNumMetricsPerMsg,
		protocolType:     defaultProtocolType,
		renameLabels:     make(map[string]string),
		allowedLabels:    make(map[string]string),
		customizeLabel:   nil,
	}

	if len(uri.Host) > 0 {
		config.host = uri.Host
	}
	opts := uri.Query()
	if len(opts["numMetricsPerMsg"]) >= 1 {
		val, err := strconv.Atoi(opts["numMetricsPerMsg"][0])
		if err != nil {
			return config, fmt.Errorf("failed to parse `numMetricsPerMsg` field - %v", err)
		}
		config.numMetricsPerMsg = val
	}
	if len(opts["protocolType"]) >= 1 {
		config.protocolType = strings.ToLower(opts["protocolType"][0])
	}
	if len(opts["prefix"]) >= 1 {
		config.prefix = opts["prefix"][0]
	}
	if len(opts["renameLabels"]) >= 1 {
		renameLabels := strings.Split(opts["renameLabels"][0], ",")
		for _, renameLabel := range renameLabels {
			kv := strings.SplitN(renameLabel, ":", 2)
			config.renameLabels[kv[0]] = kv[1]
		}
	}
	if len(opts["allowedLabels"]) >= 1 {
		allowedLabels := strings.Split(opts["allowedLabels"][0], ",")
		for _, allowedLabel := range allowedLabels {
			config.allowedLabels[allowedLabel] = allowedLabel
		}
	}
	labelStyle := DefaultLabelStyle
	if len(opts["labelStyle"]) >= 1 {
		switch opts["labelStyle"][0] {
		case "lowerCamelCase":
			labelStyle = SnakeToLowerCamel
		case "upperCamelCase":
			labelStyle = SnakeToUpperCamel
		default:
			glog.Errorf("invalid labelStyle - %s", opts["labelStyle"][0])
		}
	}
	labelCustomizer := LabelCustomizer{config.renameLabels, labelStyle}
	config.customizeLabel = labelCustomizer.Customize
	glog.Infof("statsd metrics sink using configuration : %+v", config)
	return config, nil
}

func (sink *statsdSink) ExportData(dataBatch *core.DataBatch) {
	sink.Lock()
	defer sink.Unlock()

	var metrics []string
	var tmpstr string
	var err error
	allowAllLabels := len(sink.config.allowedLabels) == 0
	for _, metricSet := range dataBatch.MetricSets {
		var metricSetLabels map[string]string
		if allowAllLabels {
			metricSetLabels = metricSet.Labels
		} else {
			metricSetLabels = make(map[string]string)
			for k, v := range metricSet.Labels {
				_, allowed := sink.config.allowedLabels[k]
				if allowed {
					metricSetLabels[k] = v
				}
			}
		}
		for metricName, metricValue := range metricSet.MetricValues {
			tmpstr, err = sink.formatter.Format(sink.config.prefix, metricName, metricSetLabels, sink.config.customizeLabel, metricValue)
			if err != nil {
				glog.Errorf("statsd metrics sink - failed to format metrics : %s", err.Error())
				continue
			}
			metrics = append(metrics, tmpstr)
		}
		for _, metric := range metricSet.LabeledMetrics {
			labels := make(map[string]string)
			for k, v := range metricSetLabels {
				labels[k] = v
			}
			for k, v := range metric.Labels {
				_, allowed := sink.config.allowedLabels[k]
				if allowed || allowAllLabels {
					labels[k] = v
				}
			}
			tmpstr, err = sink.formatter.Format(sink.config.prefix, metric.Name, labels, sink.config.customizeLabel, metric.MetricValue)
			if err != nil {
				glog.Errorf("statsd metrics sink - failed to format labeled metrics : %v", err)
				continue
			}
			metrics = append(metrics, tmpstr)
		}
	}
	glog.V(5).Infof("Sending metrics --- %s", metrics)
	err = sink.client.send(metrics)
	if err != nil {
		glog.Errorf("statsd metrics sink - failed to send some metrics : %v", err)
	}
}

func (sink *statsdSink) Name() string {
	return "StatsD Sink"
}

func (sink *statsdSink) Stop() {
	glog.V(2).Info("statsd metrics sink is stopping")
	sink.client.close()
}

func NewStatsdSinkWithClient(uri *url.URL, client statsdClient) (sink core.DataSink, err error) {
	config, err := getConfig(uri)
	if err != nil {
		return nil, err
	}
	formatter, err := NewFormatter(config.protocolType)
	if err != nil {
		return nil, err
	}
	glog.V(2).Info("statsd metrics sink is created")
	return &statsdSink{
		config:    config,
		formatter: formatter,
		client:    client,
	}, nil
}

func NewStatsdSink(uri *url.URL) (sink core.DataSink, err error) {
	config, err := getConfig(uri)
	if err != nil {
		return nil, err
	}
	client, err := NewStatsdClient(config.host, config.numMetricsPerMsg)
	if err != nil {
		return nil, err
	}
	return NewStatsdSinkWithClient(uri, client)
}
