// Copyright 2014 Google Inc. All Rights Reserved.
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

package integration

import (
	"flag"
	"fmt"
	"os"
	"io/ioutil"
	"path"
	"strings"
	"testing"
	"time"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kube_api_v1beta1 "github.com/GoogleCloudPlatform/kubernetes/pkg/api/v1beta1"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	influxdb "github.com/influxdb/influxdb/client"
	"github.com/stretchr/testify/require"
)

var (
	influxdbController          = "monitoring-influxGrafanaController"
	heapsterController          = "monitoring-heapsterController"
	targetTags                  = "kubernetes-minion"
	heapsterFirewallRule        = "heapster-e2e"
	influxdbLabels              = map[string]string{"name": "influxGrafana"}
	heapsterLabels              = map[string]string{"name": "heapster"}
	kubeVersions                = flag.String("kube_versions", "0.7.2,0.8.1", "Comma separated list of kube versions to test against")
	heapsterManifestFile        = flag.String("heapster_controller", "../deploy/heapster-controller.yaml", "Path to heapster replication controller file.")
	influxdbGrafanaManifestFile = flag.String("influxdb_grafana_controller", "../deploy/influxdb-grafana-controller.yaml", "Path to Influxdb-Grafana replication controller file.")
	influxdbServiceFile         = flag.String("influxdb_service", "../deploy/influxdb-service.yaml", "Path to Inlufxdb service file.")
	heapsterImage               = flag.String("heapster_image", "vish/heapster:e2e_test", "heapster docker image that needs to be tested.")
	influxdbImage               = flag.String("influxdb_image", "vish/heapster_influxdb:e2e_test", "influxdb docker image that needs to be tested.")
	grafanaImage                = flag.String("grafana_image", "vish/heapster_grafana:e2e_test", "grafana docker image that needs to be tested.")
	maxInfluxdbRetries          = 5
)

func replaceImages(inputFile, outputBaseDir string, containerNameImageMap map[string]string) (string, error) {
	input, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return "", err
	}

	rc := kube_api_v1beta1.ReplicationController{}
	if err := yaml.Unmarshal(input, &rc); err != nil {
		return "", err
	}
	for i, container := range rc.DesiredState.PodTemplate.DesiredState.Manifest.Containers {
		if newImage, ok := containerNameImageMap[container.Name]; ok {
			rc.DesiredState.PodTemplate.DesiredState.Manifest.Containers[i].Image = newImage
		}
	}

	output, err := yaml.Marshal(rc)
	if err != nil {
		return "", err
	}
	outFile := path.Join(outputBaseDir, path.Base(inputFile))

	return outFile, ioutil.WriteFile(outFile, output, 0644)
}

func waitUntilPodRunning(fm kubeFramework, podLabels map[string]string, timeout time.Duration) (string, error) {
	podsInterface := fm.Client().Pods(kube_api.NamespaceDefault)
	for i := 0; i < int(timeout/time.Second); i++ {
		selector := labels.Set(podLabels).AsSelector()
		podList, err := podsInterface.List(selector)
		if err != nil {
			glog.V(1).Info(err)
			return "", err
		}
		if len(podList.Items) != 1 {
			glog.V(1).Info(err)
			return "", fmt.Errorf("found %d pod with labels %v", len(podList.Items), podLabels)
		}
		podSpec := podList.Items[0]
		glog.V(2).Infof("%+v", podSpec)
		if podSpec.Status.Phase == kube_api.PodRunning {
			return podSpec.Status.HostIP, nil
		}
		time.Sleep(time.Second)
	}
	return "", fmt.Errorf("pod not in running state after %d seconds", timeout/time.Second)
}

func updateReplicas(fm kubeFramework, name string, count int) error {
	rcInterface := fm.Client().ReplicationControllers(kube_api.NamespaceDefault)
	controller, err := rcInterface.Get(name)
	if err == nil {
		controller.Spec.Replicas = 0
		if _, rerr := rcInterface.Update(controller); rerr != nil {
			return rerr
		}
	}
	glog.Errorf("updateReplicas failed for controller %+v with error: %q", controller, err)
	return err
}

func deletePods(fm kubeFramework) {
	if out, err := fm.RunKubecfgCmd("resize", influxdbController, "0"); err != nil {
		glog.Errorf("failed to bring down the number of replicas for influxdb: %q - %v", out, err)
	}
	if out, err := fm.RunKubecfgCmd("resize", heapsterController, "0"); err != nil {
		glog.Errorf("failed to bring down the number of replicas for influxdb: %q - %v", out, err)
	}
	_, _ = fm.RunKubectlCmd("delete", "-f", *heapsterManifestFile)
	_, _ = fm.RunKubectlCmd("delete", "-f", *influxdbGrafanaManifestFile)
	_, _ = fm.RunKubectlCmd("delete", "-f", *influxdbServiceFile)
}

func TestHeapsterInfluxDBWorks(t *testing.T) {
	var fm kubeFramework
	var err error

	tempDir, err := ioutil.TempDir("", "deploy")
	require.NoError(t, err, "failed to create temporary directory")
	defer os.RemoveAll(tempDir)

	kubeVersionsList := strings.Split(*kubeVersions, ",")
	for _, kubeVersion := range kubeVersionsList {
		fm, err = newKubeFramework(t, kubeVersion)
		require.NoError(t, err, "failed to create kube framework")

		// add firewall rules.
		if !fm.FirewallRuleExists(heapsterFirewallRule) {
			err := fm.AddFirewallRule(heapsterFirewallRule, []string{"tcp:8086"}, targetTags)
			require.NoError(t, err, "failed to create firewall rule")
		}

		// Delete any pre-existing heapster pods
		deletePods(fm)

		// Add test docker image
		heapsterManifest, err := replaceImages(*heapsterManifestFile, tempDir, map[string]string{"heapster": *heapsterImage})
		require.NoError(t, err, "failed to add test heapster docker image to manifest")
		influxdbGrafanaManifest, err := replaceImages(*influxdbGrafanaManifestFile, tempDir, map[string]string{"influxdb": *influxdbImage, "grafana": *grafanaImage})
		require.NoError(t, err, "failed to add test influxdb and grafana docker images to manifest")

		// Create heapster pods
		out, err := fm.RunKubectlCmd("create", "-f", influxdbGrafanaManifest)
		require.NoError(t, err, "failed to create Influxdb-grafana pod ", out)
		out, err = fm.RunKubectlCmd("create", "-f", *influxdbServiceFile)
		require.NoError(t, err, "failed to create Influxdb service ", out)
		out, err = fm.RunKubectlCmd("create", "-f", heapsterManifest)
		require.NoError(t, err, "failed to create heapster pod ", out)

		glog.V(1).Info("waiting for pods to be running")
		influxdbHostIP, err := waitUntilPodRunning(fm, influxdbLabels, 10*time.Minute)
		require.NoError(t, err, "influxdb pod is not running")
		_, err = waitUntilPodRunning(fm, heapsterLabels, 1*time.Minute)
		require.NoError(t, err, "heapster pod is not running")

		glog.V(1).Infof("checking if data exists in influxdb running at %s", influxdbHostIP)
		config := &influxdb.ClientConfig{
			Host: influxdbHostIP + ":8086",
			// TODO(vishh): Infer username and pw from the Pod spec.
			Username: "root",
			Password: "root",
			Database: "k8s",
			IsSecure: false,
		}
		influxdbClient, err := influxdb.NewClient(config)
		require.NoError(t, err, "failed to create influxdb client")

		var series []*influxdb.Series
		for i := 0; i < maxInfluxdbRetries; i++ {
			series, err = influxdbClient.Query("select * from stats limit 1", influxdb.Second)
			if err == nil {
				break
			}
			time.Sleep(30 * time.Second)
		}
		require.NoError(t, err, "failed to query data from 'stats' table in Influxdb")
		require.NotEmpty(t, series, "'stats' table does not contain any data")

		for i := 0; i < maxInfluxdbRetries; i++ {
			series, err = influxdbClient.Query("select * from machine limit 1", influxdb.Second)
			if err == nil {
				break
			}
			time.Sleep(30 * time.Second)
		}
		require.NoError(t, err, "failed to query data from 'machine' table in Influxdb")
		require.NotEmpty(t, series, "'machine' table does not contain any data")
		glog.V(1).Info("HeapsterInfluxDB test passed")
	}
	fm.DestroyCluster()
}
