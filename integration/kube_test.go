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
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	sink_api "github.com/GoogleCloudPlatform/heapster/sinks/api"
	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kube_api_v1beta1 "github.com/GoogleCloudPlatform/kubernetes/pkg/api/v1beta1"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	influxdb "github.com/influxdb/influxdb/client"
	"github.com/stretchr/testify/require"
)

const (
	heapsterPackage    = "github.com/GoogleCloudPlatform/heapster"
	targetTags         = "kubernetes-minion"
	maxInfluxdbRetries = 5
)

var (
	kubeVersions                  = flag.String("kube_versions", "", "Comma separated list of kube versions to test against")
	heapsterControllerFile        = flag.String("heapster_controller", "../deploy/heapster-controller.yaml", "Path to heapster replication controller file.")
	influxdbGrafanaControllerFile = flag.String("influxdb_grafana_controller", "../deploy/influxdb-grafana-controller.yaml", "Path to Influxdb-Grafana replication controller file.")
	influxdbServiceFile           = flag.String("influxdb_service", "../deploy/influxdb-service.yaml", "Path to Inlufxdb service file.")
	heapsterImage                 = flag.String("heapster_image", "heapster:e2e_test", "heapster docker image that needs to be tested.")
	influxdbImage                 = flag.String("influxdb_image", "heapster_influxdb:e2e_test", "influxdb docker image that needs to be tested.")
	grafanaImage                  = flag.String("grafana_image", "heapster_grafana:e2e_test", "grafana docker image that needs to be tested.")
	namespace                     = flag.String("namespace", "default", "namespace to be used for testing")
	heapsterBuildDir              = "../deploy"
	influxdbBuildDir              = "../influx-grafana/influxdb"
	grafanaBuildDir               = "../influx-grafana/grafana"
)

func buildAndPushHeapsterImage(hostnames []string) error {
	curwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(heapsterBuildDir); err != nil {
		return err
	}
	if err := buildGoBinary(heapsterPackage); err != nil {
		return err
	}
	if err := buildDockerImage(*heapsterImage); err != nil {
		return err
	}
	for _, host := range hostnames {
		if err := copyDockerImage(*heapsterImage, host); err != nil {
			return err
		}
	}
	glog.Info("built and pushed heapster image")
	return os.Chdir(curwd)
}

func buildAndPushInfluxdbImage(hostnames []string) error {
	curwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(influxdbBuildDir); err != nil {
		return err
	}
	if err := buildDockerImage(*influxdbImage); err != nil {
		return err
	}
	for _, host := range hostnames {
		if err := copyDockerImage(*influxdbImage, host); err != nil {
			return err
		}
	}
	glog.Info("built and pushed influxdb image")
	return os.Chdir(curwd)
}

func buildAndPushGrafanaImage(hostnames []string) error {
	curwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(grafanaBuildDir); err != nil {
		return err
	}
	if err := buildDockerImage(*grafanaImage); err != nil {
		return err
	}
	for _, host := range hostnames {
		if err := copyDockerImage(*grafanaImage, host); err != nil {
			return err
		}
	}
	glog.Info("built and pushed grafana image")
	return os.Chdir(curwd)
}

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

func waitUntilPodRunning(fm kubeFramework, ns string, podLabels map[string]string, timeout time.Duration) error {
	podsInterface := fm.Client().Pods(ns)
	for i := 0; i < int(timeout/time.Second); i++ {
		selector := labels.Set(podLabels).AsSelector()
		podList, err := podsInterface.List(selector)
		if err != nil {
			glog.V(1).Info(err)
			return err
		}
		if len(podList.Items) > 0 {
			podSpec := podList.Items[0]
			glog.V(2).Infof("%+v", podSpec)
			if podSpec.Status.Phase == kube_api.PodRunning {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("pod not in running state after %d seconds", timeout/time.Second)
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
	var err error
	for _, rc := range rcs {
		if err = fm.DeleteRC(ns, rc); err != nil {
			glog.Error(err)
		}
	}

	for _, service := range services {
		if err = fm.DeleteService(ns, service); err != nil {
			glog.Error(err)
		}
	}
	if err = removeDockerImage(*heapsterImage); err != nil {
		glog.Error(err)
	}
	if err = removeDockerImage(*influxdbImage); err != nil {
		glog.Error(err)
	}
	if err = removeDockerImage(*grafanaImage); err != nil {
		glog.Error(err)
	}
	var nodes []string
	if nodes, err = fm.GetNodes(); err == nil {
		for _, node := range nodes {
			cleanupRemoteHost(node)
		}
	} else {
		glog.Errorf("failed to cleanup nodes - %v", err)
	}
}

var replicationControllers = []*kube_api.ReplicationController{}
var services = []*kube_api.Service{}
var influxdbService = ""

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
	if err = waitUntilPodRunning(fm, ns, influxdbRC.Spec.Template.Labels, 10*time.Minute); err != nil {
		return err
	}
	return waitUntilPodRunning(fm, ns, heapsterRC.Spec.Template.Labels, 1*time.Minute)
}

func queryInfluxDB(t *testing.T, client *influxdb.Client) {
	var series []*influxdb.Series
	var err error
	success := false
	for i := 0; i < maxInfluxdbRetries; i++ {
		if series, err = client.Query("list series", influxdb.Second); err == nil {
			glog.V(1).Infof("query:' list series' - output %+v from influxdb", series[0].Points)
			if len(series[0].Points) >= (len(sink_api.SupportedStatMetrics()) - 1) {
				success = true
				break
			}
		}
		glog.V(2).Infof("influxdb test case failed. Retrying")
		time.Sleep(30 * time.Second)
	}
	require.NoError(t, err, "failed to list series in Influxdb")
	require.True(t, success, "list series test case failed.")
}

func buildAndPushImages(fm kubeFramework) error {
	nodes, err := fm.GetNodes()
	if err != nil {
		return err
	}
	if err := buildAndPushHeapsterImage(nodes); err != nil {
		return err
	}
	if err := buildAndPushInfluxdbImage(nodes); err != nil {
		return err
	}
	return buildAndPushGrafanaImage(nodes)
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

		require.NoError(t, buildAndPushImages(fm), "failed to build and push images")

		// create pods and wait for them to run.
		require.NoError(t, createAndWaitForRunning(fm, *namespace))

		kubeMasterHttpClient, ok := fm.Client().Client.(*http.Client)
		require.True(t, ok, "failed to get http client to kube master.")

		glog.V(2).Infof("checking if data exists in influxdb using apiserver proxy url %q", fm.GetProxyUrlForService(services[0].Name))
		config := &influxdb.ClientConfig{
			Host: fm.GetProxyUrlForService(services[0].Name),
			// TODO(vishh): Infer username and pw from the Pod spec.
			Username:   "root",
			Password:   "root",
			Database:   "k8s",
			HttpClient: kubeMasterHttpClient,
			IsSecure:   true,
		}
		influxdbClient, err := influxdb.NewClient(config)
		require.NoError(t, err, "failed to create influxdb client")

		queryInfluxDB(t, influxdbClient)

		glog.Info("**HeapsterInfluxDB test passed**")
		deleteAll(fm, *namespace, services, replicationControllers)
	}
}
