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

package apiserver

import (
	"fmt"
	"net/url"
	"time"

	. "k8s.io/heapster/metrics/core"
	"k8s.io/heapster/metrics/sources/kubelet"

	"github.com/golang/glog"
	kube_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	kube_client "k8s.io/kubernetes/pkg/client/unversioned"
	//	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
	//	"k8s.io/kubernetes/pkg/util/wait"
)

const (
	resyncPeriod = 5 * time.Minute
)

type apiServerMetricsSource struct {
	kubeClient *kube_client.Client
	dplLister  cache.StoreToDeploymentLister
	rsLister   cache.StoreToReplicaSetLister
	rcLister   *cache.StoreToReplicationControllerLister
	dsLister   cache.StoreToDaemonSetLister
}

func NewApiServerMetricsSource(client *kube_client.Client, dplLister cache.StoreToDeploymentLister, rsLister cache.StoreToReplicaSetLister, rcLister *cache.StoreToReplicationControllerLister, dsLister cache.StoreToDaemonSetLister) MetricsSource {
	return &apiServerMetricsSource{
		kubeClient: client,
		dplLister:  dplLister,
		rsLister:   rsLister,
		rcLister:   rcLister,
		dsLister:   dsLister,
	}
}

func (this *apiServerMetricsSource) Name() string {
	return this.String()
}

func (this *apiServerMetricsSource) String() string {
	return fmt.Sprintf("kube-apiserver")
}

// getDeploymentsInfo translates the deployment Status/Spec the flattened heapster MetricSet API.
func (this *apiServerMetricsSource) getDeploymentsInfo(metrics map[string]*MetricSet) {
	dpls, err := this.dplLister.List()
	if err != nil {
		glog.Errorf("%v", err)
		return
	}

	for _, d := range dpls {
		labels := map[string]string{
			"object_name":    d.Name,
			"type":           "deployment",
			"namespace_name": d.Namespace,
		}
		// Create metrics
		metric := &MetricSet{
			Labels:         labels,
			MetricValues:   map[string]MetricValue{},
			LabeledMetrics: []LabeledMetric{},
		}
		// Add metrics
		this.addIntMetric(metric, &MetricDesiredReplicas, uint64(d.Spec.Replicas))
		this.addIntMetric(metric, &MetricScheduledReplicas, uint64(d.Status.Replicas))
		this.addIntMetric(metric, &MetricUpdatedReplicas, uint64(d.Status.UpdatedReplicas))
		this.addIntMetric(metric, &MetricAvailableReplicas, uint64(d.Status.AvailableReplicas))
		this.addIntMetric(metric, &MetricUnavailableReplicas, uint64(d.Status.UnavailableReplicas))
		metrics["deployment:"+d.Name] = metric
	}
}

// getReplicaSetsInfo translates the replicatSet Status/Spec the flattened heapster MetricSet API.
func (this *apiServerMetricsSource) getReplicaSetsInfo(metrics map[string]*MetricSet) {
	rss, err := this.rsLister.List()
	if err != nil {
		glog.Errorf("%v", err)
		return
	}

	for _, rs := range rss {
		labels := map[string]string{
			"object_name":    rs.Name,
			"type":           "replicaset",
			"namespace_name": rs.Namespace,
		}
		// Create metrics
		metric := &MetricSet{
			Labels:         labels,
			MetricValues:   map[string]MetricValue{},
			LabeledMetrics: []LabeledMetric{},
		}
		// Add metrics
		this.addIntMetric(metric, &MetricDesiredReplicas, uint64(rs.Spec.Replicas))
		this.addIntMetric(metric, &MetricScheduledReplicas, uint64(rs.Status.Replicas))
		this.addIntMetric(metric, &MetricFullyLabeledReplicas, uint64(rs.Status.FullyLabeledReplicas))
		metrics["replicaset:"+rs.Name] = metric
	}
}

// getReplicationControllersInfo translates the replicationController Status/Spec the flattened heapster MetricSet API.
func (this *apiServerMetricsSource) getReplicationControllersInfo(metrics map[string]*MetricSet) {
	rcs, err := this.rcLister.List()
	if err != nil {
		glog.Errorf("%v", err)
		return
	}

	for _, rc := range rcs {
		labels := map[string]string{
			"object_name":    rc.Name,
			"type":           "replicationcontroller",
			"namespace_name": rc.Namespace,
		}
		// Create metrics
		metric := &MetricSet{
			Labels:         labels,
			MetricValues:   map[string]MetricValue{},
			LabeledMetrics: []LabeledMetric{},
		}
		// Add metrics
		this.addIntMetric(metric, &MetricDesiredReplicas, uint64(rc.Spec.Replicas))
		this.addIntMetric(metric, &MetricScheduledReplicas, uint64(rc.Status.Replicas))
		this.addIntMetric(metric, &MetricFullyLabeledReplicas, uint64(rc.Status.FullyLabeledReplicas))
		metrics["replicationcontroller:"+rc.Name] = metric
	}
}

// getDaemonSetsInfo translates the daemonSet Status/Spec the flattened heapster MetricSet API.
func (this *apiServerMetricsSource) getDaemonSetsInfo(metrics map[string]*MetricSet) {
	dss, err := this.dsLister.List()
	if err != nil {
		glog.Errorf("%v", err)
		return
	}

	for _, ds := range dss.Items {
		labels := map[string]string{
			"object_name":    ds.Name,
			"type":           "daemonset",
			"namespace_name": ds.Namespace,
		}
		// Create metrics
		metric := &MetricSet{
			Labels:         labels,
			MetricValues:   map[string]MetricValue{},
			LabeledMetrics: []LabeledMetric{},
		}
		// Add metrics
		this.addIntMetric(metric, &MetricCurrentNumberScheduledDS, uint64(ds.Status.CurrentNumberScheduled))
		this.addIntMetric(metric, &MetricNumberMisscheduledDS, uint64(ds.Status.NumberMisscheduled))
		this.addIntMetric(metric, &MetricDesiredNumberScheduledDS, uint64(ds.Status.DesiredNumberScheduled))
		metrics["daemonset:"+ds.Name] = metric
	}
}

func (this *apiServerMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	result := &DataBatch{
		Timestamp:  time.Now(),
		MetricSets: map[string]*MetricSet{},
	}

	// Deployments
	this.getDeploymentsInfo(result.MetricSets)

	// ReplicaSets
	this.getReplicaSetsInfo(result.MetricSets)

	// ReplicationControllers
	this.getReplicationControllersInfo(result.MetricSets)

	// DaemonSets
	this.getDaemonSetsInfo(result.MetricSets)

	return result
}

// addIntMetric is a convenience method for adding the metric and value to the metric set.
func (this *apiServerMetricsSource) addIntMetric(metrics *MetricSet, metric *Metric, value uint64) {
	val := MetricValue{
		ValueType:  ValueInt64,
		MetricType: metric.Type,
		IntValue:   int64(value),
	}
	metrics.MetricValues[metric.Name] = val
}

type apiServerProvider struct {
	dplLister  cache.StoreToDeploymentLister
	rsLister   cache.StoreToReplicaSetLister
	rcLister   *cache.StoreToReplicationControllerLister
	dsLister   cache.StoreToDaemonSetLister
	kubeClient *kube_client.Client
}

func (this *apiServerProvider) GetMetricsSources() []MetricsSource {
	sources := []MetricsSource{}
	sources = append(sources, NewApiServerMetricsSource(this.kubeClient, this.dplLister, this.rsLister, this.rcLister, this.dsLister))
	return sources
}

func NewApiServerProvider(uri *url.URL) (MetricsSourceProvider, error) {
	// create clients
	kubeConfig, _, err := kubelet.GetKubeConfigs(uri)
	if err != nil {
		return nil, err
	}
	kubeClient := kube_client.NewOrDie(kubeConfig)

	// Deployment
	dplStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
	dplListWatch := &cache.ListWatch{
		ListFunc: func(options kube_api.ListOptions) (runtime.Object, error) {
			return kubeClient.Extensions().Deployments(kube_api.NamespaceAll).List(options)
		},
		WatchFunc: func(options kube_api.ListOptions) (watch.Interface, error) {
			return kubeClient.Extensions().Deployments(kube_api.NamespaceAll).Watch(options)
		},
	}
	cache.NewReflector(dplListWatch, &extensions.Deployment{}, dplStore, 0).Run()
	dplLister := cache.StoreToDeploymentLister{Store: dplStore}

	// ReplicaSet
	rsStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
	rsListWatch := &cache.ListWatch{
		ListFunc: func(options kube_api.ListOptions) (runtime.Object, error) {
			return kubeClient.Extensions().ReplicaSets(kube_api.NamespaceAll).List(options)
		},
		WatchFunc: func(options kube_api.ListOptions) (watch.Interface, error) {
			return kubeClient.Extensions().ReplicaSets(kube_api.NamespaceAll).Watch(options)
		},
	}
	cache.NewReflector(rsListWatch, &extensions.ReplicaSet{}, rsStore, 0).Run()
	rsLister := cache.StoreToReplicaSetLister{Store: rsStore}

	// ReplicationController
	lw := cache.NewListWatchFromClient(kubeClient, "replicationcontrollers", kube_api.NamespaceAll, fields.Everything())
	rcLister := &cache.StoreToReplicationControllerLister{Indexer: cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})}
	reflector := cache.NewReflector(lw, &kube_api.ReplicationController{}, rcLister.Indexer, time.Hour)
	reflector.Run()

	// DaemonSet
	dsStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
	dsListWatch := &cache.ListWatch{
		ListFunc: func(options kube_api.ListOptions) (runtime.Object, error) {
			return kubeClient.Extensions().DaemonSets(kube_api.NamespaceAll).List(options)
		},
		WatchFunc: func(options kube_api.ListOptions) (watch.Interface, error) {
			return kubeClient.Extensions().DaemonSets(kube_api.NamespaceAll).Watch(options)
		},
	}
	cache.NewReflector(dsListWatch, &extensions.DaemonSet{}, dsStore, 0).Run()
	dsLister := cache.StoreToDaemonSetLister{Store: dsStore}

	return &apiServerProvider{
		kubeClient: kubeClient,
		dplLister:  dplLister,
		rsLister:   rsLister,
		rcLister:   rcLister,
		dsLister:   dsLister,
	}, nil
}
