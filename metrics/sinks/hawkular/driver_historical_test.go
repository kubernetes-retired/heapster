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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hawkular/hawkular-client-go/metrics"
	"k8s.io/heapster/metrics/core"

	assert "github.com/stretchr/testify/require"
)

// Helpers

func testServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tenant := r.Header.Get("Hawkular-Tenant")

		rPath := r.URL.Path[18:]
		values := r.URL.Query()

		path := strings.Split(rPath, "/")
		typ := path[0]

		var b []byte
		empty := false

		// Tenants emulation
		if typ == "tenants" {
			b = []byte(`[{"id": "1CE77F8206798F1C587611A4574683C7"}, {"id": "test-heapster", "retentions": {"gauge": 5}}]`)
			// w.WriteHeader(http.StatusOK)
			// return
		}

		if typ == "metrics" {
			// TagValues emulation
			if len(path) > 1 && path[1] == "tags" {
				tags := parseTags(path[2])
				resp := make(map[string][]string)
				for k, v := range tags {
					if !strings.HasSuffix(v, "*") { // * is something we"ll add later
						resp[k] = []string{v}
						continue
					}

					switch k {
					case core.LabelPodName.Key:
						resp[core.LabelPodName.Key] = []string{"pod1", "pod2", "pod3"}
					case core.LabelHostname.Key:
						resp[core.LabelHostname.Key] = []string{"host1", "host2"}
					case core.LabelContainerName.Key:
						if tenant != "test-heapster" {
							empty = true
						} else {
							resp[core.LabelContainerName.Key] = []string{"hawkular", "heapster"}
						}
					case core.LabelPodId.Key:
						if v == "terrible*" {
							resp[descriptorTag] = []string{"terribleghost"}
						}
					case core.LabelNamespaceName.Key:
						resp[core.LabelNamespaceName.Key] = []string{"namespacey"}
					}
				}

				if !empty {
					b, _ = json.Marshal(resp)
				}
			}

			if len(values) > 0 {
				tags := parseTags(values.Get("tags"))
				// Mixed metric definition fetching emulation
				if values.Get("type") == "counter" && tags[descriptorTag] == "cpu/usage" {
					// Value: tags -> [type:cluster,descriptor_name:cpu/usage]
					if tags["type"] == "cluster" {
						b = []byte(`[{"type": "counter", "id": "heapster.test.cluster.cpu.usage", "tags": {"type": "cluster", "descriptor_name": "cpu/usage"}, "dataRetention": 7, "tenantId": "test-heapster"}]`)
					}
					// Value: tags -> [type:sys_container,nodename:node_1,container_name:container_1,descriptor_name:cpu/usage]
					// Value: tags -> [descriptor_name:cpu/usage,type:pod,namespace_name:namespace_1,pod_name:pod_name_2]
					// Value: tags -> [pod_id:pod_id_2,descriptor_name:cpu/usage,type:pod]
					if tags["type"] == "pod" && tags["pod_id"] == "pod_id_2" {
						b = []byte(`[{"type": "counter", "id": "heapster.test.podid.cpu.usage", "tags": {"type": "pod", "pod_id": "pod_id_2", "descriptor_name": "cpu/usage"}, "dataRetention": 7, "tenantId": "test-heapster"}]`)
					}
					// Value: tags -> [pod_name:pod_name_1,namespace_name:namespace_1,descriptor_name:cpu/usage,type:pod_container,container_name:container_2]
					// Value: tags -> [type:node,nodename:nodename_1,descriptor_name:cpu/usage]
					// Value: tags -> [type:pod_container,container_name:container_2,pod_id:pod_id_1,descriptor_name:cpu/usage]
					// Value: tags -> [type:ns,namespace_name:namespace_2,descriptor_name:cpu/usage]
				}
			}
		}

		if typ == "counters" {
			// Emulate metric datapoint fetching
			id := path[1]

			start, err := strconv.ParseInt(values.Get("start"), 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			end, err := strconv.ParseInt(values.Get("end"), 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if id == "stats" {
				// Emulate aggregate datapoint fetching with tags

				// PATH: counters/stats
				// Value: bucketDuration -> [3600000ms]
				// Value: end -> [1467725505218]
				// Value: percentiles -> [95,99]
				// Value: start -> [1467721905218]
				// Value: tags -> [type:cluster,descriptor_name:cpu/usage]

				tags := parseTags(values.Get("tags"))
				if tags[descriptorTag] == "cpu/usage" && tags["type"] == "pod" && tags["pod_id"] == "pod_id_2" {
					buckets := []metrics.Bucketpoint{
						metrics.Bucketpoint{
							Start:   metrics.FromUnixMilli(start),
							End:     metrics.FromUnixMilli(end),
							Min:     float64(1.0),
							Max:     float64(16.0),
							Avg:     float64(6.2),
							Median:  float64(4.0),
							Samples: 5,
							Percentiles: []metrics.Percentile{
								metrics.Percentile{
									Quantile: 95.0,
									Value:    4.0, // Not really, but for testing purposes
								},
								metrics.Percentile{
									Quantile: 99.0,
									Value:    8.0,
								},
							},
							Empty: false,
						},
					}
					b, _ = json.Marshal(buckets)
				}
			} else {
				queryType := path[2]
				if queryType == "raw" {
					// Emulate raw datapoint fetching
					if id == "heapster.test.podid.cpu.usage" {

						data := []metrics.Datapoint{
							metrics.Datapoint{
								Timestamp: metrics.FromUnixMilli(start + 1),
								Value:     3,
							},
							metrics.Datapoint{
								Timestamp: metrics.FromUnixMilli(end - 1),
								Value:     4,
							},
						}
						b, _ = json.Marshal(data)
					} else if id == "heapster.test.cluster.cpu.usage" {
						data := []metrics.Datapoint{
							metrics.Datapoint{
								Timestamp: metrics.FromUnixMilli(start + 1),
								Value:     2,
							},
						}
						b, _ = json.Marshal(data)
					}
				}
			}
		}
		if !empty && b != nil {
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
}

func parseTags(tagParam string) map[string]string {
	tags := make(map[string]string)
	for _, v := range strings.Split(tagParam, ",") {
		tagsSplit := strings.Split(v, ":")
		tags[tagsSplit[0]] = tagsSplit[1]
	}
	return tags
}

func historicKeys() []core.HistoricalKey {
	return []core.HistoricalKey{
		core.HistoricalKey{
			ObjectType:    core.MetricSetTypeSystemContainer,
			NodeName:      "node_1",
			ContainerName: "container_1",
		},
		core.HistoricalKey{
			ObjectType:    core.MetricSetTypePodContainer,
			PodId:         "pod_id_1",
			ContainerName: "container_2",
		},
		core.HistoricalKey{
			ObjectType:    core.MetricSetTypePodContainer,
			ContainerName: "container_2",
			NamespaceName: "namespace_1",
			PodName:       "pod_name_1",
		},
		core.HistoricalKey{
			ObjectType: core.MetricSetTypePod,
			PodId:      "pod_id_2",
		},
		core.HistoricalKey{
			ObjectType:    core.MetricSetTypePod,
			NamespaceName: "namespace_1",
			PodName:       "pod_name_2",
		},
		core.HistoricalKey{
			ObjectType:    core.MetricSetTypeNamespace,
			NamespaceName: "namespace_2",
		},
		core.HistoricalKey{
			ObjectType: core.MetricSetTypeNode,
			NodeName:   "nodename_1",
		},
		core.HistoricalKey{
			ObjectType:    core.MetricSetTypeCluster,
			NamespaceName: "namespace_3",
			NodeName:      "cluster_node_1",
		},
	}
}

func historicTest(t *testing.T) *hawkularSink {
	s := testServer()

	hSink, err := integSink(s.URL + "?tenant=test-heapster&labelToTenant=pod_namespace&batchSize=20&concurrencyLimit=5")
	assert.NoError(t, err)

	return hSink
}

// Public functions
func TestGetMetricNames(t *testing.T) {

	hSink := historicTest(t)

	hKey := core.HistoricalKey{
		ObjectType: core.MetricSetTypePod,
		PodId:      "terrible*",
	}

	names, err := hSink.GetMetricNames(hKey)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(names))

	hKey.PodId = "unknown*"
	names, err = hSink.GetMetricNames(hKey)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(names))
}

func TestGetNodes(t *testing.T) {
	hSink := historicTest(t)

	nodes, err := hSink.GetNodes()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nodes))
}

func TestGetMetric(t *testing.T) {
	hSink := historicTest(t)

	metricName := core.MetricCpuUsage.Name
	metricKeys := historicKeys()

	tmvs, err := hSink.GetMetric(metricName, metricKeys, time.Now().Add(-1*time.Hour), time.Now())
	assert.NoError(t, err)

	assert.Equal(t, 2, len(tmvs))
	assert.Equal(t, 1, len(tmvs[metricKeys[7]]))
	assert.Equal(t, 2, len(tmvs[metricKeys[3]]))
}

func TestGetAggregation(t *testing.T) {
	hSink := historicTest(t)

	metricName := core.MetricCpuUsage.Name
	metricKeys := historicKeys()

	tavs, err := hSink.GetAggregation(metricName, core.MultiTypedAggregations, metricKeys, time.Now().Add(-1*time.Hour), time.Now(), time.Hour)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(tavs))
	assert.Equal(t, 1, len(tavs[metricKeys[3]]))
	assert.Equal(t, uint64(5), *tavs[metricKeys[3]][0].Count)
}

func TestGetPodsFromNamespace(t *testing.T) {
	hSink := historicTest(t)

	pods, err := hSink.GetPodsFromNamespace("namespacey")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(pods))
}

func TestGetSystemContainersFromNode(t *testing.T) {
	hSink := historicTest(t)

	sysContainers, err := hSink.GetSystemContainersFromNode("host1")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(sysContainers))
}

func TestGetNamespaces(t *testing.T) {
	s := testServer()

	hSink, err := integSink(s.URL + "?tenant=test-heapster&labelToTenant=pod_namespace&batchSize=20&concurrencyLimit=5")
	assert.NoError(t, err)

	// From tenants
	namespaces, err := hSink.GetNamespaces()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(namespaces))

	// From tags
	hSink, err = integSink(s.URL + "?tenant=test-heapster")
	assert.NoError(t, err)
	namespaces, err = hSink.GetNamespaces()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(namespaces))
}

// Internal functions

func TestDatapointToMetricValue(t *testing.T) {
	hSink := dummySink()

	mt := metrics.Gauge

	ts := time.Now()
	val := float64(1.45)

	dp := metrics.Datapoint{
		Timestamp: ts,
		Value:     val,
	}

	tmv, err := hSink.datapointToMetricValue(&dp, &mt)
	assert.NoError(t, err)

	assert.Equal(t, float32(val), tmv.MetricValue.FloatValue)
	assert.Equal(t, core.MetricGauge, tmv.MetricType)
	assert.Equal(t, core.ValueFloat, tmv.MetricValue.ValueType)

	mt = metrics.Counter
	vali := int64(3)
	dp = metrics.Datapoint{
		Timestamp: ts,
		Value:     vali,
	}

	tmv, err = hSink.datapointToMetricValue(&dp, &mt)
	assert.NoError(t, err)

	assert.Equal(t, vali, tmv.MetricValue.IntValue)
	assert.Equal(t, core.MetricCumulative, tmv.MetricType)
	assert.Equal(t, core.ValueInt64, tmv.MetricValue.ValueType)
}

func TestBucketPointToAggregationValue(t *testing.T) {
	hSink := dummySink()

	start := time.Now()

	bp := metrics.Bucketpoint{
		Start:   start,
		End:     start.Add(time.Second),
		Min:     float64(1.1),
		Max:     float64(2.1),
		Median:  float64(1.6),
		Avg:     float64(1.6),
		Empty:   false,
		Samples: uint64(3),
		Percentiles: []metrics.Percentile{
			metrics.Percentile{
				Quantile: float64(95.0),
				Value:    float64(1.6),
			},
		},
	}

	mt := metrics.Gauge
	bucketSize := time.Minute

	tav, err := hSink.bucketPointToAggregationValue(&bp, &mt, core.MultiTypedAggregations, bucketSize)
	assert.NoError(t, err)

	assert.Equal(t, len(core.MultiTypedAggregations), len(tav.Aggregations))
	assert.Equal(t, uint64(3), *tav.Count)
	assert.Equal(t, start, tav.Timestamp) // Agreed value on the PR review
}

func TestKeyToTags(t *testing.T) {
	hSink := dummySink()

	hKey := core.HistoricalKey{
		ObjectType: core.MetricSetTypePod,
		PodId:      "terrible",
	}

	mmap := hSink.keyToTags(&hKey)

	assert.Equal(t, core.MetricSetTypePod, mmap[core.LabelMetricSetType.Key])
	assert.Equal(t, "terrible", mmap[core.LabelPodId.Key])
}

func TestIsNamespaceTenant(t *testing.T) {
	c, err := integSink("http://doesnotmatter:8080/?tenant=test-heapster&labelToTenant=pod_namespace")
	assert.NoError(t, err)

	assert.True(t, c.isNamespaceTenant())

	c, err = integSink("http://doesnotmatter:8080/?tenant=test-heapster")
	assert.NoError(t, err)

	assert.False(t, c.isNamespaceTenant())
}
