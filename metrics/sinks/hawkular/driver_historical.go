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

package hawkular

import (
	"fmt"
	"sync"
	"time"

	"github.com/hawkular/hawkular-client-go/metrics"
	"k8s.io/heapster/metrics/core"
)

// Historical returns the historical data access interface for this sink
func (h *hawkularSink) Historical() core.HistoricalSource {
	return h
}

// GetMetric retrieves the given metric for one or more objects (specified by metricKeys) of
// the same type, within the given time interval
func (h *hawkularSink) GetMetric(metricName string, metricKeys []core.HistoricalKey, start, end time.Time) (map[core.HistoricalKey][]core.TimestampedMetricValue, error) {
	return h.GetLabeledMetric(metricName, nil, metricKeys, start, end)
}

// GetLabeledMetric retrieves the given labeled metric.  Otherwise, it functions identically to GetMetric.
func (h *hawkularSink) GetLabeledMetric(metricName string, labels map[string]string, metricKeys []core.HistoricalKey, start, end time.Time) (map[core.HistoricalKey][]core.TimestampedMetricValue, error) {
	typ := h.metricNameToHawkularType(metricName)

	resLock := &sync.Mutex{}
	res := make(map[core.HistoricalKey][]core.TimestampedMetricValue, len(metricKeys))

	errChan := make(chan error, len(metricKeys))
	wg := &sync.WaitGroup{}

	for _, keyM := range metricKeys {
		wg.Add(1)
		go func(key core.HistoricalKey) {
			defer wg.Done()
			tags := h.keyToTags(&key)
			tags[descriptorTag] = metricName

			if labels != nil && len(labels) > 0 {
				for k, v := range labels {
					tags[k] = v
				}
			}

			o := []metrics.Modifier{metrics.Filters(metrics.TagsFilter(tags), metrics.TypeFilter(typ))}
			if h.isNamespaceTenant() && key.NamespaceName != "" && key.ObjectType != core.MetricSetTypeCluster {
				o = append(o, metrics.Tenant(key.NamespaceName))
			}

			// Remove in favor of using ReadRaw with filters to reduce the amount of queries to HWKMETRICS,
			// but that API is still considered unstable in current release
			metricDefs, err := h.client.Definitions(o...)
			if err != nil {
				errChan <- err
				return
			}

			if len(metricDefs) > 1 {
				errChan <- fmt.Errorf("Given metric query (metricName=%q, key=%q) did not result in unique metricId, found %d results", metricName, key, len(metricDefs))
				return
			} else if len(metricDefs) < 1 {
				return
			}

			filters := make([]metrics.Filter, 0, 2)

			// Without start & endTime, the default is requesting 8 hours to the past
			if !end.IsZero() {
				filters = append(filters, metrics.EndTimeFilter(end))
			}

			if !start.IsZero() {
				filters = append(filters, metrics.StartTimeFilter(start))
			}

			datapoints, err := h.client.ReadRaw(metricDefs[0].Type, metricDefs[0].ID, metrics.Filters(filters...))
			if err != nil {
				errChan <- err
				return
			}

			if len(datapoints) > 0 {
				metricValues := make([]core.TimestampedMetricValue, 0, len(datapoints))

				for _, datapoint := range datapoints {
					value, err := h.datapointToMetricValue(datapoint, &typ)
					if err != nil {
						errChan <- err
						return
					}
					metricValues = append(metricValues, value)
				}

				resLock.Lock()
				res[key] = metricValues
				resLock.Unlock()
			}

		}(keyM)
	}
	wg.Wait()
	close(errChan)

	// Check for errors first
	for e := range errChan {
		return nil, e
	}

	return res, nil
}

// GetAggregation fetches the given aggregations for one or more objects (specified by metricKeys) of
// the same type, within the given time interval, calculated over a series of buckets
func (h *hawkularSink) GetAggregation(metricName string, aggregations []core.AggregationType, metricKeys []core.HistoricalKey, start, end time.Time, bucketSize time.Duration) (map[core.HistoricalKey][]core.TimestampedAggregationValue, error) {
	return h.GetLabeledAggregation(metricName, nil, aggregations, metricKeys, start, end, bucketSize)
}

// GetLabeledAggregation fetches a the given aggregations for a labeled metric instead of a normal metric.
// Otherwise, it functions identically to GetAggregation.
func (h *hawkularSink) GetLabeledAggregation(metricName string, labels map[string]string, aggregations []core.AggregationType, metricKeys []core.HistoricalKey, start, end time.Time, bucketSize time.Duration) (map[core.HistoricalKey][]core.TimestampedAggregationValue, error) {
	typ := h.metricNameToHawkularType(metricName)

	resLock := &sync.Mutex{}
	res := make(map[core.HistoricalKey][]core.TimestampedAggregationValue, len(metricKeys))

	errChan := make(chan error, len(metricKeys))
	wg := &sync.WaitGroup{}

	// While we could read everything in a single query to Hawkular-Metrics, we could not reliably parse the results back to HistoricalKey
	for _, keyM := range metricKeys {
		wg.Add(1)
		go func(key core.HistoricalKey) {
			defer wg.Done()
			tags := h.keyToTags(&key)
			tags[descriptorTag] = metricName

			if labels != nil && len(labels) > 0 {
				for k, v := range labels {
					tags[k] = v
				}
			}

			o := make([]metrics.Modifier, 0, 2)
			if h.isNamespaceTenant() && key.NamespaceName != "" && key.ObjectType != core.MetricSetTypeCluster {
				o = append(o, metrics.Tenant(key.NamespaceName))
			}

			filters := make([]metrics.Filter, 0, 5)
			filters = append(filters, metrics.TagsFilter(tags))

			if bucketSize != 0 {
				filters = append(filters, metrics.BucketsDurationFilter(bucketSize))
			} else {
				// No bucketSize defined, lets request a single bucket over all the data
				filters = append(filters, metrics.BucketsFilter(1))
			}

			// Without start & endTime, the default is requesting 8 hours to the past
			if !end.IsZero() {
				filters = append(filters, metrics.EndTimeFilter(end))
			}

			if !start.IsZero() {
				filters = append(filters, metrics.StartTimeFilter(start))
			}

			// No need to request percentile 50 as that equals median
			filters = append(filters, metrics.PercentilesFilter([]float64{95.0, 99.0}))

			o = append(o, metrics.Filters(filters...))
			buckets, err := h.client.ReadBuckets(typ, o...)
			if err != nil {
				errChan <- err
				return
			}

			if len(buckets) > 0 {
				aggregationValues := make([]core.TimestampedAggregationValue, 0, len(buckets))

				for _, bucketPoint := range buckets {
					value, err := h.bucketPointToAggregationValue(bucketPoint, &typ, aggregations, bucketSize)
					if err != nil {
						errChan <- err
						return
					}

					aggregationValues = append(aggregationValues, value)
				}
				resLock.Lock()
				res[key] = aggregationValues
				resLock.Unlock()
			}

		}(keyM)
	}
	wg.Wait()
	close(errChan)

	// Check for errors first
	for e := range errChan {
		return nil, e
	}

	return res, nil
}

// GetMetricNames retrieves the available metric names for the given object
func (h *hawkularSink) GetMetricNames(metricKey core.HistoricalKey) ([]string, error) {
	tags := h.keyToTags(&metricKey)
	tags[descriptorTag] = "*"

	var r map[string][]string
	var err error

	if h.isNamespaceTenant() && metricKey.NamespaceName != "" {
		r, err = h.client.TagValues(tags, metrics.Tenant(metricKey.NamespaceName))
	} else {
		r, err = h.client.TagValues(tags)
	}

	if r != nil {
		return r[descriptorTag], err
	}
	return nil, err
}

// GetNodes retrieves the list of nodes in the cluster
func (h *hawkularSink) GetNodes() ([]string, error) {
	tags := make(map[string]string)
	tags[core.LabelHostname.Key] = "*"
	tags[core.LabelMetricSetType.Key] = core.MetricSetTypeNode
	r, err := h.client.TagValues(tags) // Assume these are stored in the system tenant
	if r != nil {
		return r[core.LabelHostname.Key], err
	}
	return nil, err
}

// GetPodsFromNamespace retrieves the list of pods in a given namespace
func (h *hawkularSink) GetPodsFromNamespace(namespace string) ([]string, error) {
	tags := make(map[string]string)
	tags[core.LabelPodName.Key] = "*"
	tags[core.LabelMetricSetType.Key] = core.MetricSetTypePod

	var r map[string][]string
	var err error

	if h.isNamespaceTenant() {
		r, err = h.client.TagValues(tags, metrics.Tenant(namespace))
	} else {
		tags[core.LabelNamespaceName.Key] = namespace
		r, err = h.client.TagValues(tags)
	}

	if r != nil {
		return r[core.LabelPodName.Key], err
	}
	return nil, err
}

// GetSystemContainersFromNode retrieves the list of free containers for a given node
func (h *hawkularSink) GetSystemContainersFromNode(node string) ([]string, error) {
	tags := make(map[string]string)
	tags[core.LabelContainerName.Key] = "*"
	tags[core.LabelHostname.Key] = node
	tags[core.LabelMetricSetType.Key] = core.MetricSetTypeSystemContainer
	r, err := h.client.TagValues(tags) // I assume these are stored at the system tenant
	if r != nil {
		return r[core.LabelContainerName.Key], err
	}
	return nil, err
}

// GetNamespaces retrieves the list of namespaces in the cluster
func (h *hawkularSink) GetNamespaces() ([]string, error) {
	// Fetch tenants, if the labelToTenantId is set (and includes namespace label)
	if h.isNamespaceTenant() {
		tds, err := h.client.Tenants()
		if err != nil {
			return nil, err
		}
		ns := make([]string, 0, len(tds))
		for _, td := range tds {
			ns = append(ns, td.ID)
		}
		return ns, nil
	}

	return h.getLabelTagValues(core.LabelNamespaceName.Key)
}

// Internal functions

func (h *hawkularSink) getLabelTagValues(labelKey string, o ...metrics.Modifier) ([]string, error) {
	tags := make(map[string]string)
	tags[labelKey] = "*"
	r, err := h.client.TagValues(tags, o...)
	if r != nil {
		return r[labelKey], err
	}
	return nil, err
}

// isNamespaceTenant tries to detect if namespace labels are used as target tenant. This is in use on Openshift as default (and Hawkular images)
// if this returns true, the namespace information should be used when sending a query to the Hawkular, and only cluster level statistics are
// fetched from the default tenant
func (h *hawkularSink) isNamespaceTenant() bool {
	// This could be a bit problematic.. are we really storing the right info?
	if h.labelTenant != "" && (h.labelTenant == core.LabelPodNamespace.Key || h.labelTenant == core.LabelPodNamespaceUID.Key) {
		return true
	}
	return false
}

func (h *hawkularSink) bucketPointToAggregationValue(bp *metrics.Bucketpoint, mt *metrics.MetricType, aggregations []core.AggregationType, bucketSize time.Duration) (core.TimestampedAggregationValue, error) {
	aggregationValues := core.TimestampedAggregationValue{
		Timestamp:  bp.Start,
		BucketSize: bucketSize,
		AggregationValue: core.AggregationValue{
			Count: &bp.Samples,
		},
	}

	aggs := make(map[core.AggregationType]core.MetricValue, len(aggregations))

	for _, a := range aggregations {
		var val float64
		switch a {
		case core.AggregationTypeAverage:
			val = bp.Avg
		case core.AggregationTypeMaximum:
			val = bp.Max
		case core.AggregationTypeMinimum:
			val = bp.Min
		case core.AggregationTypeMedian:
			val = bp.Median
		case core.AggregationTypePercentile50:
			val = bp.Median // Percentile 50 and Median are the same thing
		case core.AggregationTypePercentile95:
			for _, v := range bp.Percentiles {
				if v.Quantile == 95.0 {
					val = v.Value
				}
			}
		case core.AggregationTypePercentile99:
			for _, v := range bp.Percentiles {
				if v.Quantile == 99.0 {
					val = v.Value
				}
			}
		}

		mv := core.MetricValue{
			MetricType: core.MetricGauge,
			ValueType:  core.ValueFloat,
			FloatValue: float32(val),
		}
		f, err := metrics.ConvertToFloat64(val)
		if err != nil {
			return aggregationValues, err
		}
		mv.FloatValue = float32(f)

		aggs[a] = mv
	}

	aggregationValues.AggregationValue.Aggregations = aggs
	return aggregationValues, nil
}

func (h *hawkularSink) datapointToMetricValue(dp *metrics.Datapoint, mt *metrics.MetricType) (core.TimestampedMetricValue, error) {
	mv := core.MetricValue{
		MetricType: hawkularTypeToHeapsterType(*mt),
	}

	tmv := core.TimestampedMetricValue{
		Timestamp: dp.Timestamp,
	}

	switch *mt {
	case metrics.Counter:
		mv.ValueType = core.ValueInt64
		if v, ok := dp.Value.(int64); ok {
			mv.IntValue = int64(v)
		}
	case metrics.Gauge:
		mv.ValueType = core.ValueFloat
		f, err := metrics.ConvertToFloat64(dp.Value)
		if err != nil {
			return tmv, err
		}
		mv.FloatValue = float32(f)
		// if v, ok := dp.Value.(float64); ok {

		// }
	}
	tmv.MetricValue = mv

	return tmv, nil
}

func (h *hawkularSink) metricNameToHawkularType(metricName string) metrics.MetricType {
	if metric, found := core.AllMetricsMapping[metricName]; found {
		return heapsterTypeToHawkularType(metric.Type)
	}
	return metrics.Gauge
}

func (h *hawkularSink) keyToTags(metricKey *core.HistoricalKey) map[string]string {
	tags := make(map[string]string)
	tags[core.LabelMetricSetType.Key] = metricKey.ObjectType

	switch metricKey.ObjectType {
	case core.MetricSetTypeSystemContainer:
		tags[core.LabelNodename.Key] = metricKey.NodeName
		tags[core.LabelContainerName.Key] = metricKey.ContainerName
	case core.MetricSetTypePodContainer:
		tags[core.LabelContainerName.Key] = metricKey.ContainerName

		if metricKey.PodId != "" {
			tags[core.LabelPodId.Key] = metricKey.PodId
		} else {
			tags[core.LabelPodName.Key] = metricKey.PodName
			tags[core.LabelNamespaceName.Key] = metricKey.NamespaceName
		}
	case core.MetricSetTypePod:
		if metricKey.PodId != "" {
			tags[core.LabelPodId.Key] = metricKey.PodId
		} else {
			tags[core.LabelNamespaceName.Key] = metricKey.NamespaceName
			tags[core.LabelPodName.Key] = metricKey.PodName
		}
	case core.MetricSetTypeNamespace:
		tags[core.LabelNamespaceName.Key] = metricKey.NamespaceName
	case core.MetricSetTypeNode:
		tags[core.LabelNodename.Key] = metricKey.NodeName
	case core.MetricSetTypeCluster:
		// System tenant
	}
	return tags
}
