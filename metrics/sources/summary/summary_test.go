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

package summary

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	util "k8s.io/client-go/util/testing"
	"k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sources/kubelet"
	"k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
)

const (
	// Offsets from seed value in generated container stats.
	offsetCPUUsageCores = iota
	offsetCPUUsageCoreSeconds
	offsetMemPageFaults
	offsetMemMajorPageFaults
	offsetMemUsageBytes
	offsetMemRSSBytes
	offsetMemWorkingSetBytes
	offsetNetRxBytes
	offsetNetRxErrors
	offsetNetTxBytes
	offsetNetTxErrors
	offsetFsUsed
	offsetFsCapacity
	offsetFsAvailable
)

const (
	seedNode           = 0
	seedRuntime        = 100
	seedKubelet        = 200
	seedMisc           = 300
	seedPod0           = 1000
	seedPod0Container0 = 2000
	seedPod0Container1 = 2001
	seedPod1           = 3000
	seedPod1Container  = 4000
	seedPod2           = 5000
	seedPod2Container0 = 6000
	seedPod2Container1 = 7000
	seedPod3Container0 = 9000
	seedPod4           = 10000
	seedPod4Container0 = 11000
)

const (
	namespace0 = "test0"
	namespace1 = "test1"

	pName0 = "pod0"
	pName1 = "pod1"
	pName2 = "pod0" // ensure pName2 conflicts with pName0, but is in a different namespace
	pName3 = "pod2"
	pName4 = "pod4" // Regression test for #1838

	cName00 = "c0"
	cName01 = "c1"
	cName10 = "c0"      // ensure cName10 conflicts with cName02, but is in a different pod
	cName20 = "c1"      // ensure cName20 conflicts with cName01, but is in a different pod + namespace
	cName21 = "runtime" // ensure that runtime containers are not renamed
	cName30 = "c3"
	cName40 = "c4" // Running, with cpu / memory stats
	cName41 = "c4" // Terminated, has no CPU / Memory stats
	cName42 = "c4" // Terminated, has blank CPU / Memory stats
)

var (
	availableFsBytes = uint64(1130)
	usedFsBytes      = uint64(13340)
	totalFsBytes     = uint64(2153)
	freeInode        = uint64(10440)
	usedInode        = uint64(103520)
	totalInode       = uint64(103620)
	scrapeTime       = time.Now()
	startTime        = time.Now().Add(-time.Minute)
)

var nodeInfo = NodeInfo{
	NodeName:       "test",
	HostName:       "test-hostname",
	HostID:         "1234567890",
	KubeletVersion: "1.2",
}

type fakeSource struct {
	scraped bool
}

func (f *fakeSource) Name() string { return "fake" }
func (f *fakeSource) ScrapeMetrics(start, end time.Time) (*core.DataBatch, error) {
	f.scraped = true
	return nil, nil
}

func testingSummaryMetricsSource() *summaryMetricsSource {
	return &summaryMetricsSource{
		node:          nodeInfo,
		kubeletClient: &kubelet.KubeletClient{},
	}
}

func TestDecodeSummaryMetrics(t *testing.T) {

	ms := testingSummaryMetricsSource()
	summary := v1alpha1.Summary{
		Node: v1alpha1.NodeStats{
			NodeName:  nodeInfo.NodeName,
			StartTime: metav1.NewTime(startTime),
			CPU:       genTestSummaryCPU(seedNode),
			Memory:    genTestSummaryMemory(seedNode),
			Network:   genTestSummaryNetwork(seedNode),
			SystemContainers: []v1alpha1.ContainerStats{
				genTestSummaryContainer(v1alpha1.SystemContainerKubelet, seedKubelet),
				genTestSummaryContainer(v1alpha1.SystemContainerRuntime, seedRuntime),
				genTestSummaryContainer(v1alpha1.SystemContainerMisc, seedMisc),
			},
			Fs: genTestSummaryFsStats(seedNode),
		},
		Pods: []v1alpha1.PodStats{{
			PodRef: v1alpha1.PodReference{
				Name:      pName0,
				Namespace: namespace0,
			},
			StartTime: metav1.NewTime(startTime),
			Network:   genTestSummaryNetwork(seedPod0),
			Containers: []v1alpha1.ContainerStats{
				genTestSummaryContainer(cName00, seedPod0Container0),
				genTestSummaryContainer(cName01, seedPod0Container1),
				genTestSummaryTerminatedContainer(cName00, seedPod0Container0),
			},
		}, {
			PodRef: v1alpha1.PodReference{
				Name:      pName1,
				Namespace: namespace0,
			},
			StartTime: metav1.NewTime(startTime),
			Network:   genTestSummaryNetwork(seedPod1),
			Containers: []v1alpha1.ContainerStats{
				genTestSummaryContainer(cName10, seedPod1Container),
			},
			VolumeStats: []v1alpha1.VolumeStats{{
				Name:    "A",
				FsStats: *genTestSummaryFsStats(seedPod1),
			}, {
				Name:    "B",
				FsStats: *genTestSummaryFsStats(seedPod1),
			}},
		}, {
			PodRef: v1alpha1.PodReference{
				Name:      pName2,
				Namespace: namespace1,
			},
			StartTime: metav1.NewTime(startTime),
			Network:   genTestSummaryNetwork(seedPod2),
			Containers: []v1alpha1.ContainerStats{
				genTestSummaryContainer(cName20, seedPod2Container0),
				genTestSummaryContainer(cName21, seedPod2Container1),
			},
		}, {
			PodRef: v1alpha1.PodReference{
				Name:      pName3,
				Namespace: namespace0,
			},
			Containers: []v1alpha1.ContainerStats{
				genTestSummaryContainer(cName30, seedPod3Container0),
			},
			VolumeStats: []v1alpha1.VolumeStats{{
				Name: "C",
				FsStats: v1alpha1.FsStats{
					AvailableBytes: &availableFsBytes,
					UsedBytes:      &usedFsBytes,
					CapacityBytes:  &totalFsBytes,
					InodesFree:     &freeInode,
					InodesUsed:     &usedInode,
					Inodes:         &totalInode,
				},
			},
			},
		}, {
			PodRef: v1alpha1.PodReference{
				Name:      pName4,
				Namespace: namespace0,
			},
			StartTime: metav1.NewTime(startTime),
			Network:   genTestSummaryNetwork(seedPod4),
			Containers: []v1alpha1.ContainerStats{
				genTestSummaryContainer(cName40, seedPod4Container0),
				genTestSummaryTerminatedContainerNoStats(cName41),
				genTestSummaryTerminatedContainerBlankStats(cName41),
			},
		}},
	}

	containerFs := []string{"/", "logs"}
	expectations := []struct {
		key     string
		setType string
		seed    int64
		cpu     bool
		memory  bool
		network bool
		fs      []string
	}{{
		key:     core.NodeKey(nodeInfo.NodeName),
		setType: core.MetricSetTypeNode,
		seed:    seedNode,
		cpu:     true,
		memory:  true,
		network: true,
		fs:      []string{"/"},
	}, {
		key:     core.NodeContainerKey(nodeInfo.NodeName, "kubelet"),
		setType: core.MetricSetTypeSystemContainer,
		seed:    seedKubelet,
		cpu:     true,
		memory:  true,
	}, {
		key:     core.NodeContainerKey(nodeInfo.NodeName, "docker-daemon"),
		setType: core.MetricSetTypeSystemContainer,
		seed:    seedRuntime,
		cpu:     true,
		memory:  true,
	}, {
		key:     core.NodeContainerKey(nodeInfo.NodeName, "system"),
		setType: core.MetricSetTypeSystemContainer,
		seed:    seedMisc,
		cpu:     true,
		memory:  true,
	}, {
		key:     core.PodKey(namespace0, pName0),
		setType: core.MetricSetTypePod,
		seed:    seedPod0,
		network: true,
	}, {
		key:     core.PodKey(namespace0, pName1),
		setType: core.MetricSetTypePod,
		seed:    seedPod1,
		network: true,
		fs:      []string{"Volume:A", "Volume:B"},
	}, {
		key:     core.PodKey(namespace1, pName2),
		setType: core.MetricSetTypePod,
		seed:    seedPod2,
		network: true,
	}, {
		key:     core.PodKey(namespace0, pName4),
		setType: core.MetricSetTypePod,
		seed:    seedPod4,
		network: true,
	}, {
		key:     core.PodContainerKey(namespace0, pName0, cName00),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod0Container0,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName0, cName01),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod0Container1,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName1, cName10),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod1Container,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace1, pName2, cName20),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod2Container0,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace1, pName2, cName21),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod2Container1,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName3, cName30),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod3Container0,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}, {
		key:     core.PodContainerKey(namespace0, pName4, cName40),
		setType: core.MetricSetTypePodContainer,
		seed:    seedPod4Container0,
		cpu:     true,
		memory:  true,
		fs:      containerFs,
	}}

	metrics := ms.decodeSummary(&summary)
	for _, e := range expectations {
		m, ok := metrics[e.key]
		if !assert.True(t, ok, "missing metric %q", e.key) {
			continue
		}
		assert.Equal(t, m.Labels[core.LabelMetricSetType.Key], e.setType, e.key)
		assert.Equal(t, m.CreateTime, startTime, e.key)
		assert.Equal(t, m.ScrapeTime, scrapeTime, e.key)
		if e.cpu {
			checkIntMetric(t, m, e.key, core.MetricCpuUsage, e.seed+offsetCPUUsageCoreSeconds)
		}
		if e.memory {
			checkIntMetric(t, m, e.key, core.MetricMemoryUsage, e.seed+offsetMemUsageBytes)
			checkIntMetric(t, m, e.key, core.MetricMemoryWorkingSet, e.seed+offsetMemWorkingSetBytes)
			checkIntMetric(t, m, e.key, core.MetricMemoryRSS, e.seed+offsetMemRSSBytes)
			checkIntMetric(t, m, e.key, core.MetricMemoryPageFaults, e.seed+offsetMemPageFaults)
			checkIntMetric(t, m, e.key, core.MetricMemoryMajorPageFaults, e.seed+offsetMemMajorPageFaults)
		}
		if e.network {
			checkIntMetric(t, m, e.key, core.MetricNetworkRx, e.seed+offsetNetRxBytes)
			checkIntMetric(t, m, e.key, core.MetricNetworkRxErrors, e.seed+offsetNetRxErrors)
			checkIntMetric(t, m, e.key, core.MetricNetworkTx, e.seed+offsetNetTxBytes)
			checkIntMetric(t, m, e.key, core.MetricNetworkTxErrors, e.seed+offsetNetTxErrors)
		}
		for _, label := range e.fs {
			checkFsMetric(t, m, e.key, label, core.MetricFilesystemAvailable, e.seed+offsetFsAvailable)
			checkFsMetric(t, m, e.key, label, core.MetricFilesystemLimit, e.seed+offsetFsCapacity)
			checkFsMetric(t, m, e.key, label, core.MetricFilesystemUsage, e.seed+offsetFsUsed)
		}
		delete(metrics, e.key)
	}

	// Verify volume information labeled metrics
	var volumeInformationMetricsKey = core.PodKey(namespace0, pName3)
	var mappedVolumeStats = map[string]int64{}
	for _, labeledMetric := range metrics[volumeInformationMetricsKey].LabeledMetrics {
		assert.True(t, strings.HasPrefix("Volume:C", labeledMetric.Labels["resource_id"]))
		mappedVolumeStats[labeledMetric.Name] = labeledMetric.IntValue
	}

	assert.True(t, mappedVolumeStats["filesystem/available"] == int64(availableFsBytes))
	assert.True(t, mappedVolumeStats["filesystem/usage"] == int64(usedFsBytes))
	assert.True(t, mappedVolumeStats["filesystem/limit"] == int64(totalFsBytes))

	delete(metrics, volumeInformationMetricsKey)

	for k, v := range metrics {
		assert.Fail(t, "unexpected metric", "%q: %+v", k, v)
	}
}

func genTestSummaryTerminatedContainer(name string, seed int) v1alpha1.ContainerStats {
	return v1alpha1.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime.Add(-time.Minute)),
		CPU:       genTestSummaryZeroCPU(seed),
		Memory:    genTestSummaryZeroMemory(seed),
		Rootfs:    genTestSummaryFsStats(seed),
		Logs:      genTestSummaryFsStats(seed),
	}
}

func genTestSummaryTerminatedContainerNoStats(name string) v1alpha1.ContainerStats {
	return v1alpha1.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime.Add(-time.Minute)),
	}
}

func genTestSummaryTerminatedContainerBlankStats(name string) v1alpha1.ContainerStats {
	return v1alpha1.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime.Add(-time.Minute)),
		CPU:       genTestSummaryBlankCPU(),
		Memory:    genTestSummaryBlankMemory(),
	}
}

func genTestSummaryContainer(name string, seed int) v1alpha1.ContainerStats {
	return v1alpha1.ContainerStats{
		Name:      name,
		StartTime: metav1.NewTime(startTime),
		CPU:       genTestSummaryCPU(seed),
		Memory:    genTestSummaryMemory(seed),
		Rootfs:    genTestSummaryFsStats(seed),
		Logs:      genTestSummaryFsStats(seed),
	}
}

func genTestSummaryZeroCPU(seed int) *v1alpha1.CPUStats {
	cpu := v1alpha1.CPUStats{
		Time:                 metav1.NewTime(scrapeTime),
		UsageNanoCores:       uint64Val(seed, -seed),
		UsageCoreNanoSeconds: uint64Val(seed, offsetCPUUsageCoreSeconds),
	}
	*cpu.UsageCoreNanoSeconds *= uint64(time.Millisecond.Nanoseconds())
	return &cpu
}

func genTestSummaryCPU(seed int) *v1alpha1.CPUStats {
	cpu := v1alpha1.CPUStats{
		Time:                 metav1.NewTime(scrapeTime),
		UsageNanoCores:       uint64Val(seed, offsetCPUUsageCores),
		UsageCoreNanoSeconds: uint64Val(seed, offsetCPUUsageCoreSeconds),
	}
	*cpu.UsageNanoCores *= uint64(time.Millisecond.Nanoseconds())
	return &cpu
}

func genTestSummaryBlankCPU() *v1alpha1.CPUStats {
	return &v1alpha1.CPUStats{
		Time: metav1.NewTime(scrapeTime),
	}
}

func genTestSummaryZeroMemory(seed int) *v1alpha1.MemoryStats {
	return &v1alpha1.MemoryStats{
		Time:            metav1.NewTime(scrapeTime),
		UsageBytes:      uint64Val(seed, offsetMemUsageBytes),
		WorkingSetBytes: uint64Val(seed, offsetMemWorkingSetBytes),
		RSSBytes:        uint64Val(seed, -seed),
		PageFaults:      uint64Val(seed, offsetMemPageFaults),
		MajorPageFaults: uint64Val(seed, offsetMemMajorPageFaults),
	}
}

func genTestSummaryMemory(seed int) *v1alpha1.MemoryStats {
	return &v1alpha1.MemoryStats{
		Time:            metav1.NewTime(scrapeTime),
		UsageBytes:      uint64Val(seed, offsetMemUsageBytes),
		WorkingSetBytes: uint64Val(seed, offsetMemWorkingSetBytes),
		RSSBytes:        uint64Val(seed, offsetMemRSSBytes),
		PageFaults:      uint64Val(seed, offsetMemPageFaults),
		MajorPageFaults: uint64Val(seed, offsetMemMajorPageFaults),
	}
}

func genTestSummaryBlankMemory() *v1alpha1.MemoryStats {
	return &v1alpha1.MemoryStats{
		Time: metav1.NewTime(scrapeTime),
	}
}

func genTestSummaryNetwork(seed int) *v1alpha1.NetworkStats {
	return &v1alpha1.NetworkStats{
		Time:     metav1.NewTime(scrapeTime),
		RxBytes:  uint64Val(seed, offsetNetRxBytes),
		RxErrors: uint64Val(seed, offsetNetRxErrors),
		TxBytes:  uint64Val(seed, offsetNetTxBytes),
		TxErrors: uint64Val(seed, offsetNetTxErrors),
	}
}

func genTestSummaryFsStats(seed int) *v1alpha1.FsStats {
	return &v1alpha1.FsStats{
		AvailableBytes: uint64Val(seed, offsetFsAvailable),
		CapacityBytes:  uint64Val(seed, offsetFsCapacity),
		UsedBytes:      uint64Val(seed, offsetFsUsed),
	}
}

// Convenience function for taking the address of a uint64 literal.
func uint64Val(seed, offset int) *uint64 {
	val := uint64(seed + offset)
	return &val
}

func checkIntMetric(t *testing.T, metrics *core.MetricSet, key string, metric core.Metric, value int64) {
	m, ok := metrics.MetricValues[metric.Name]
	if !assert.True(t, ok, "missing %q:%q", key, metric.Name) {
		return
	}
	assert.Equal(t, value, m.IntValue, "%q:%q", key, metric.Name)
}

func checkFsMetric(t *testing.T, metrics *core.MetricSet, key, label string, metric core.Metric, value int64) {
	for _, m := range metrics.LabeledMetrics {
		if m.Name == metric.Name && m.Labels[core.LabelResourceID.Key] == label {
			assert.Equal(t, value, m.IntValue, "%q:%q[%s]", key, metric.Name, label)
			return
		}
	}
	assert.Fail(t, "missing filesystem metric", "%q:[%q]:%q", key, metric.Name, label)
}

func TestScrapeSummaryMetrics(t *testing.T) {
	summary := v1alpha1.Summary{
		Node: v1alpha1.NodeStats{
			NodeName:  nodeInfo.NodeName,
			StartTime: metav1.NewTime(startTime),
		},
	}
	data, err := json.Marshal(&summary)
	require.NoError(t, err)

	server := httptest.NewServer(&util.FakeHandler{
		StatusCode:   200,
		ResponseBody: string(data),
		T:            t,
	})
	defer server.Close()

	ms := testingSummaryMetricsSource()
	split := strings.SplitN(strings.Replace(server.URL, "http://", "", 1), ":", 2)
	ms.node.IP = split[0]
	ms.node.Port, err = strconv.Atoi(split[1])
	require.NoError(t, err)

	res, err := ms.ScrapeMetrics(time.Now(), time.Now())
	assert.Nil(t, err, "scrape error")
	assert.Equal(t, res.MetricSets["node:test"].Labels[core.LabelMetricSetType.Key], core.MetricSetTypeNode)
}
