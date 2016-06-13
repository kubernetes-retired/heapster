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

package v1

import (
	"fmt"
	"io"
	"net/http"
	"time"

	restful "github.com/emicklei/go-restful"
	prompb "github.com/prometheus/client_model/go"
	promfmt "github.com/prometheus/common/expfmt"
	authinfo "k8s.io/kubernetes/pkg/auth/user"

	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/util"
	"k8s.io/heapster/metrics/util/metrics"
)

func (a *PushApi) RegisterFormatHandlers(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/api/v1/push").
		Doc("Root endpoint of the push metrics target").
		Consumes("*/*").
		Produces(restful.MIME_JSON)
	// The /prometheus/ endpoint is normal prometheus push metrics destination
	ws.Route(ws.POST("/prometheus").
		To(metrics.InstrumentRouteFunc("prometheusPushRequest", a.prometheusPushRequest)).
		Doc("Push metrics in prometheus (text or protobuf) format").
		Operation("prometheusPushRequest"))

	// These routes are just like the /prometheus/ endpoint, but exist for
	// quasi-compatibility with Prometheus push gateway
	ws.Route(ws.POST("/prometheus/metrics/jobs/{producer}").
		To(metrics.InstrumentRouteFunc("prometheusPushRequest", a.prometheusPushRequest)).
		Doc("Push metrics in prometheus (text or protobuf) format").
		Operation("prometheusPushRequest"))
	ws.Route(ws.POST("/prometheus/metrics/job/{producer}").
		To(metrics.InstrumentRouteFunc("prometheusPushRequest", a.prometheusPushRequest)).
		Doc("Push metrics in prometheus (text or protobuf) format").
		Operation("prometheusPushRequest"))
	container.Add(ws)
}

// ingestPrometheusMetrics takes a set of headers and body content (plus a source name), and
// extract and processes the body, converting it from a Prometheus metrics format (specified
// in the headers) into a Heapster DataBatch.
func ingestPrometheusMetrics(user string, headers http.Header, body io.Reader) (*core.DataBatch, int, error) {
	metricsFormat := promfmt.ResponseFormat(headers)
	if metricsFormat == promfmt.FmtUnknown {
		return nil, http.StatusUnsupportedMediaType, fmt.Errorf("Unable to determine format of submitted Prometheus metrics")
	}

	decoder := promfmt.NewDecoder(body, metricsFormat)
	ingester := &dataBatchIngester{
		nameFormat: fmt.Sprintf("custom/%s/%%s", user),
		batch: &core.DataBatch{
			Timestamp:  nowFunc(),
			MetricSets: map[string]*core.MetricSet{},
		},
		defaultTimestamp: nowFunc(),
	}

	if err := ingester.IngestAll(decoder); err != nil {
		actualErr := fmt.Errorf("Unable to process submitted Prometheus metrics: %v", err)
		return nil, http.StatusBadRequest, actualErr
	}

	return ingester.batch, 0, nil
}

// prometheusPushRequest processes a request to push metrics in the prometheus format, converting
// them into internal Heapster form and storing them in the PushSource
func (a *PushApi) prometheusPushRequest(req *restful.Request, resp *restful.Response) {
	userInfo := req.Attribute(util.UserAttributeName).(authinfo.Info)
	userName := userInfo.GetName()
	// Double check here just to make sure we don't have a bad prefix
	if userName == "" {
		resp.WriteError(http.StatusUnauthorized, fmt.Errorf("Metrics must be submitted as a specific user"))
		return
	}

	pathUser := req.PathParameter("producer")
	if pathUser != "" && pathUser != userName {
		resp.WriteError(http.StatusUnauthorized, fmt.Errorf("Metrics must be submitted as the authenticated user"))
	}

	batch, errStatus, err := ingestPrometheusMetrics(userName, req.Request.Header, req.Request.Body)
	if err != nil {
		resp.WriteError(errStatus, err)
		return
	}

	// TODO: do we need to do this ourselves?
	req.Request.Body.Close()

	a.pushSource.PushMetrics(batch, userName)

	resp.WriteHeader(http.StatusAccepted)
}

// dataBatchIngester is a Prometheus ingester for use with a processor which
// ingests into a Heapster DataBatch
type dataBatchIngester struct {
	batch            *core.DataBatch
	nameFormat       string
	defaultTimestamp time.Time
}

func (ing *dataBatchIngester) ingestNormalMetricEntry(metricType core.MetricType, familyName string, metric *prompb.Metric, value float64) error {
	key, setLabels, normalLabels, err := extractKeyAndFilterLabels(metric.Label)
	if err != nil {
		return err
	}

	metricSet, wasPresent := ing.batch.MetricSets[key]
	if !wasPresent {
		metricSet = &core.MetricSet{
			MetricValues:   make(map[string]core.MetricValue),
			Labels:         setLabels,
			LabeledMetrics: nil,
		}

		if metric.TimestampMs != nil {
			metricSet.ScrapeTime = time.Unix(0, *metric.TimestampMs*1000000)
		} else {
			metricSet.ScrapeTime = ing.defaultTimestamp
		}

		ing.batch.MetricSets[key] = metricSet
	}

	// NB: all prometheus values are float64 values
	val := core.MetricValue{
		// WHY?  WHY ARE WE USING FLOAT32 BUT INT64?
		FloatValue: float32(value),
		ValueType:  core.ValueFloat,
		MetricType: metricType,
	}

	metricName := fmt.Sprintf(ing.nameFormat, familyName)

	// check to see if this metric needs to be added as a labeled metric
	if len(normalLabels) != 0 {
		labeledMetric := core.LabeledMetric{
			Name:        metricName,
			Labels:      normalLabels,
			MetricValue: val,
		}
		metricSet.LabeledMetrics = append(metricSet.LabeledMetrics, labeledMetric)
		return nil
	}

	metricSet.MetricValues[metricName] = val

	return nil
}

func (ing *dataBatchIngester) ingestCumulative(family *prompb.MetricFamily) error {
	familyName := family.GetName()

	for _, metricEntry := range family.Metric {
		if metricEntry.Counter == nil {
			continue
		}

		if err := ing.ingestNormalMetricEntry(core.MetricCumulative, familyName, metricEntry, metricEntry.Counter.GetValue()); err != nil {
			return err
		}
	}

	return nil
}

func (ing *dataBatchIngester) ingestGauge(family *prompb.MetricFamily) error {
	familyName := family.GetName()

	for _, metricEntry := range family.Metric {
		if metricEntry.Gauge == nil {
			continue
		}

		if err := ing.ingestNormalMetricEntry(core.MetricGauge, familyName, metricEntry, metricEntry.Gauge.GetValue()); err != nil {
			return err
		}
	}

	return nil
}

func (ing *dataBatchIngester) ingestUntyped(family *prompb.MetricFamily) error {
	familyName := family.GetName()

	for _, metricEntry := range family.Metric {
		if metricEntry.Untyped == nil {
			continue
		}

		// assume untyped is a gauge
		if err := ing.ingestNormalMetricEntry(core.MetricGauge, familyName, metricEntry, metricEntry.Untyped.GetValue()); err != nil {
			return err
		}
	}

	return nil
}

func (ing *dataBatchIngester) IngestMetricFamily(family *prompb.MetricFamily) error {
	switch family.GetType() {
	case prompb.MetricType_COUNTER:
		return ing.ingestCumulative(family)
	case prompb.MetricType_GAUGE:
		return ing.ingestGauge(family)
	case prompb.MetricType_UNTYPED:
		return ing.ingestUntyped(family)
	case prompb.MetricType_SUMMARY:
		return fmt.Errorf("unsupported metric type %q encountered", "Summary")
	case prompb.MetricType_HISTOGRAM:
		return fmt.Errorf("unsupported metric type %q encountered", "Histogram")
	default:
		return fmt.Errorf("unknown metric type identifier %v encountered", family.GetType())
	}
}

func (ing *dataBatchIngester) IngestAll(decoder promfmt.Decoder) error {
	metricFamily := &prompb.MetricFamily{}
	var err error
	for {
		err = decoder.Decode(metricFamily)
		if err != nil {
			break
		}
		ing.IngestMetricFamily(metricFamily)
	}

	if err == io.EOF {
		// EOF isn't actually an error
		err = nil
	}

	return err

}

// extractKeyAndFilterLabels extracts a key (as defined in ms_keys.go) from the given
// set of labels, and splits the remaining labels into those appropriate for a metric set
// referred to by that key and any other normal labels.
func extractKeyAndFilterLabels(labelPairs []*prompb.LabelPair) (key string, setLabels map[string]string, normalLabels map[string]string, err error) {
	setLabels = make(map[string]string)
	normalLabels = make(map[string]string)

	var (
		hasNS   = false
		hasPod  = false
		hasCont = false
		hasNode = false

		ns   string
		pod  string
		cont string
		node string
	)

	for _, labelPair := range labelPairs {
		labelName := labelPair.GetName()
		switch labelName {
		case PushLabelNamespace:
			hasNS = true
			ns = labelPair.GetValue()
			setLabels[core.LabelNamespaceName.Key] = ns
			continue
		case PushLabelPod:
			hasPod = true
			pod = labelPair.GetValue()
			setLabels[core.LabelPodName.Key] = pod
			continue
		case PushLabelContainer:
			hasCont = true
			cont = labelPair.GetValue()
			setLabels[core.LabelContainerName.Key] = cont
			continue
		case PushLabelNode:
			hasNode = true
			node = labelPair.GetValue()
			setLabels[core.LabelNodename.Key] = node
			continue
		}

		if _, ok := allLabels[labelName]; ok {
			setLabels[labelName] = labelPair.GetValue()
			continue
		}

		normalLabels[labelName] = labelPair.GetValue()
	}

	switch {
	case hasNS && hasPod && hasCont:
		setLabels[core.LabelMetricSetType.Key] = core.MetricSetTypePodContainer
		key = core.PodContainerKey(ns, pod, cont)
	case hasNS && hasPod:
		setLabels[core.LabelMetricSetType.Key] = core.MetricSetTypePod
		key = core.PodKey(ns, pod)
	case hasNS:
		setLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeNamespace
		key = core.NamespaceKey(ns)
	case hasNode && hasCont:
		setLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeSystemContainer
		key = core.NodeContainerKey(node, cont)
	case hasNode:
		setLabels[core.LabelMetricSetType.Key] = core.MetricSetTypeNode
		key = core.NodeKey(node)
	case !hasNode && !hasNS && !hasPod && !hasCont:
		key = core.ClusterKey()
	default:
		return "", nil, nil, fmt.Errorf("invalid label configuration for specifying metric key: must have labels namespace[+pod[+container]] or node[+container], or none of the above (for cluster metrics)")
	}

	return key, setLabels, normalLabels, nil
}

const (
	PushLabelNamespace = "namespace"
	PushLabelPod       = "pod"
	PushLabelContainer = "container"
	PushLabelNode      = "node"
)

var allLabels = map[string]struct{}{
	core.LabelNodename.Key: {},
	core.LabelHostname.Key: {},
	core.LabelHostID.Key:   {},

	// TODO: core.LabelPodNamespace?
	core.LabelNamespaceName.Key: {},

	core.LabelPodId.Key:   {},
	core.LabelPodName.Key: {},

	core.LabelContainerName.Key: {},
}
