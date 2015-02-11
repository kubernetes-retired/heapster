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
	"io/ioutil"
	"os"
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

const (
	targetTags           = "kubernetes-minion"
	heapsterFirewallRule = "heapster-e2e"
	maxInfluxdbRetries   = 5
)

var (
	kubeVersions                  = flag.String("kube_versions", "", "Comma separated list of kube versions to test against")
	heapsterControllerFile        = flag.String("heapster_controller", "../deploy/heapster-controller.yaml", "Path to heapster replication controller file.")
	influxdbGrafanaControllerFile = flag.String("influxdb_grafana_controller", "../deploy/influxdb-grafana-controller.yaml", "Path to Influxdb-Grafana replication controller file.")
	influxdbServiceFile           = flag.String("influxdb_service", "../deploy/influxdb-service.yaml", "Path to Inlufxdb service file.")
	heapsterImage                 = flag.String("heapster_image", "vish/heapster:e2e_test", "heapster docker image that needs to be tested.")
	influxdbImage                 = flag.String("influxdb_image", "vish/heapster_influxdb:e2e_test", "influxdb docker image that needs to be tested.")
	grafanaImage                  = flag.String("grafana_image", "vish/heapster_grafana:e2e_test", "grafana docker image that needs to be tested.")
	namespace                     = flag.String("namespace", "default", "namespace to be used for testing")
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

func waitUntilPodRunning(fm kubeFramework, ns string, podLabels map[string]string, timeout time.Duration) (string, error) {
	podsInterface := fm.Client().Pods(ns)
	for i := 0; i < int(timeout/time.Second); i++ {
		selector := labels.Set(podLabels).AsSelector()
		podList, err := podsInterface.List(selector)
		if err != nil {
			glog.V(1).Info(err)
			return "", err
		}
		if len(podList.Items) > 0 {
			podSpec := podList.Items[0]
			glog.V(2).Infof("%+v", podSpec)
			if podSpec.Status.Phase == kube_api.PodRunning {
				return podSpec.Status.HostIP, nil
			}
		}
		time.Sleep(time.Second)
	}
	return "", fmt.Errorf("pod not in running state after %d seconds", timeout/time.Second)
}

func createAll(fm kubeFramework, ns string, services []*kube_api.Service, rcs []*kube_api.ReplicationController) error {
	for _, rc := range rcs {
		if err := fm.CreateRC(ns, rc); err != nil {
			return err
		}
	}

	for _, service := range services {
		if err := fm.CreateService(ns, service); err != nil {
			return err
		}
	}
	return nil
}

func deleteAll(fm kubeFramework, ns string, services []*kube_api.Service, rcs []*kube_api.ReplicationController) {
	for _, rc := range rcs {
		if err := fm.DeleteRC(ns, rc); err != nil {
			glog.Error(err)
		}
	}

	for _, service := range services {
		if err := fm.DeleteService(ns, service); err != nil {
			glog.Error(err)
		}
	}
}

var replicationControllers = []*kube_api.ReplicationController{}
var services = []*kube_api.Service{}
var influxdbNodeIP = ""

func createAndWaitForRunning(fm kubeFramework, ns string) error {
	// Add test docker image
	heapsterRC, err := fm.ParseRC(*heapsterControllerFile)
	if err != nil {
		return fmt.Errorf("failed to parse heapster controller - %v", err)
	}
	heapsterRC.Spec.Template.Spec.Containers[0].Image = *heapsterImage
	replicationControllers = append(replicationControllers, heapsterRC)

	influxdbRC, err := fm.ParseRC(*influxdbGrafanaControllerFile)
	if err != nil {
		return fmt.Errorf("failed to parse influxdb controller - %v", err)
	}

	for _, cont := range influxdbRC.Spec.Template.Spec.Containers {
		if strings.Contains(cont.Name, "grafana") {
			cont.Image = *grafanaImage
		} else if strings.Contains(cont.Name, "influxdb") {
			cont.Image = *influxdbImage
		}
	}
	replicationControllers = append(replicationControllers, influxdbRC)

	influxdbService, err := fm.ParseService(*influxdbServiceFile)
	if err != nil {
		return fmt.Errorf("failed to parse influxdb service - %v", err)
	}
	services = append(services, influxdbService)

	deleteAll(fm, ns, services, replicationControllers)
	if err = createAll(fm, ns, services, replicationControllers); err != nil {
		return err
	}

	glog.V(1).Info("waiting for pods to be running")
	influxdbNodeIP, err = waitUntilPodRunning(fm, ns, influxdbRC.Spec.Template.Labels, 10*time.Minute)
	if err != nil {
		return err
	}
	_, err = waitUntilPodRunning(fm, ns, heapsterRC.Spec.Template.Labels, 1*time.Minute)
	return err
}

func TestHeapsterInfluxDBWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kubernetes integration test.")
	}
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

		// create pods and wait for them to run.
		require.NoError(t, createAndWaitForRunning(fm, *namespace))

		glog.V(1).Infof("checking if data exists in influxdb running at %s", influxdbNodeIP)
		config := &influxdb.ClientConfig{
			// Use master proxy to access influxdb Service.
			Host: influxdbNodeIP + ":8086",
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
		glog.Info("HeapsterInfluxDB test passed")
		deleteAll(fm, *namespace, services, replicationControllers)
	}
}
