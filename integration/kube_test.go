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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kube_api_v1beta3 "github.com/GoogleCloudPlatform/kubernetes/pkg/api/v1beta3"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/golang/glog"
	influxdb "github.com/influxdb/influxdb/client"
	"github.com/stretchr/testify/require"
)

const (
	kTestZone            = "us-central1-b"
	targetTags           = "kubernetes-minion"
	sleepBetweenAttempts = 5 * time.Second
	testTimeout          = 5 * time.Minute
	heapsterBuildDir     = "../deploy/docker"
	influxdbBuildDir     = "../influxdb"
	grafanaBuildDir      = "../grafana"
	makefile             = "../Makefile"
	podlistQuery         = "select distinct(pod_id) from /cpu.*/"
	nodelistQuery        = "select distinct(hostname) from /cpu.*/"
	eventsQuery          = "select distinct(value) from \"log/events\""
)

var (
	kubeVersions                  = flag.String("kube_versions", "", "Comma separated list of kube versions to test against. By default will run the test against an existing cluster")
	heapsterControllerFile        = flag.String("heapster_controller", "../deploy/kube-config/influxdb/heapster-controller.json", "Path to heapster replication controller file.")
	influxdbGrafanaControllerFile = flag.String("influxdb_grafana_controller", "../deploy/kube-config/influxdb/influxdb-grafana-controller.json", "Path to Influxdb-Grafana replication controller file.")
	influxdbServiceFile           = flag.String("influxdb_service", "../deploy/kube-config/influxdb/influxdb-service.json", "Path to Inlufxdb service file.")
	heapsterImage                 = flag.String("heapster_image", "heapster:e2e_test", "heapster docker image that needs to be tested.")
	influxdbImage                 = flag.String("influxdb_image", "heapster_influxdb:e2e_test", "influxdb docker image that needs to be tested.")
	grafanaImage                  = flag.String("grafana_image", "heapster_grafana:e2e_test", "grafana docker image that needs to be tested.")
	namespace                     = flag.String("namespace", "default", "namespace to be used for testing")
	avoidBuild                    = flag.Bool("nobuild", false, "When true, a new heapster docker image will not be created and pushed to test cluster nodes.")
)

func buildAndPushHeapsterImage(hostnames []string) error {
	curwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(heapsterBuildDir); err != nil {
		return err
	}
	if err := make("build", path.Join(curwd, makefile)); err != nil {
		return err
	}
	if err := buildDockerImage(*heapsterImage); err != nil {
		return err
	}
	for _, host := range hostnames {
		if err := copyDockerImage(*heapsterImage, host, kTestZone); err != nil {
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
		if err := copyDockerImage(*influxdbImage, host, kTestZone); err != nil {
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
		if err := copyDockerImage(*grafanaImage, host, kTestZone); err != nil {
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

	rc := kube_api_v1beta3.ReplicationController{}
	if err := json.Unmarshal(input, &rc); err != nil {
		return "", err
	}
	for i, container := range rc.Spec.Template.Spec.Containers {
		if newImage, ok := containerNameImageMap[container.Name]; ok {
			rc.Spec.Template.Spec.Containers[i].Image = newImage
		}
	}

	output, err := json.Marshal(rc)
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
	if *avoidBuild {
		return
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
			host := strings.Split(node, ".")[0]
			cleanupRemoteHost(host, kTestZone)
		}
	} else {
		glog.Errorf("failed to cleanup nodes - %v", err)
	}
}

var replicationControllers = []*kube_api.ReplicationController{}
var services = []*kube_api.Service{}

func createAndWaitForRunning(fm kubeFramework, ns string) error {
	// Add test docker image
	heapsterRC, err := fm.ParseRC(*heapsterControllerFile)
	if err != nil {
		return fmt.Errorf("failed to parse heapster controller - %v", err)
	}
	heapsterRC.Spec.Template.Spec.Containers[0].Image = *heapsterImage
	heapsterRC.Spec.Template.Spec.Containers[0].ImagePullPolicy = kube_api.PullNever
	// increase logging level
	heapsterRC.Spec.Template.Spec.Containers[0].Env = append(heapsterRC.Spec.Template.Spec.Containers[0].Env, kube_api.EnvVar{Name: "FLAGS", Value: "--vmodule=*=3"})
	glog.V(3).Infof("Heapster RC: %+v", heapsterRC)
	glog.V(3).Infof("Heapster Pod: %+v", heapsterRC.Spec.Template)
	glog.V(3).Infof("Heapster Container: %+v", heapsterRC.Spec.Template.Spec.Containers[0])

	replicationControllers = append(replicationControllers, heapsterRC)

	influxdbRC, err := fm.ParseRC(*influxdbGrafanaControllerFile)
	if err != nil {
		return fmt.Errorf("failed to parse influxdb controller - %v", err)
	}

	glog.V(3).Infof("Influxdb RC: %+v", influxdbRC)
	glog.V(3).Infof("Influxdb Pod: %+v", influxdbRC.Spec.Template)
	for _, cont := range influxdbRC.Spec.Template.Spec.Containers {
		glog.V(3).Infof("%s Container: %+v", cont.Name, cont)
		cont.ImagePullPolicy = kube_api.PullNever
		if strings.Contains(cont.Name, "grafana") {
			cont.Image = *grafanaImage
		} else if strings.Contains(cont.Name, "influxdb") {
			cont.Image = *influxdbImage
		}
		for idx := range cont.Ports {
			cont.Ports[idx].HostPort = cont.Ports[idx].ContainerPort
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

func buildAndPushImages(fm kubeFramework) error {
	if *avoidBuild {
		return nil
	}
	nodes, err := fm.GetNodes()
	if err != nil {
		return err
	}
	hostnames := []string{}
	for _, node := range nodes {
		hostnames = append(hostnames, strings.Split(node, ".")[0])
	}
	if err := buildAndPushHeapsterImage(hostnames); err != nil {
		return err
	}
	if err := buildAndPushInfluxdbImage(hostnames); err != nil {
		return err
	}
	return buildAndPushGrafanaImage(hostnames)
}

func getInfluxdbData(c *influxdb.Client, query string) (map[string]bool, error) {
	series, err := c.Query(query, influxdb.Second)
	if err != nil {
		return nil, err
	}
	if len(series) != 1 {
		return nil, fmt.Errorf("expected only one series from Influxdb for query %q. Got %+v", query, series)
	}
	if len(series[0].GetColumns()) != 2 {
		return nil, fmt.Errorf("Expected two columns for query %q. Found %v", query, series[0].GetColumns())
	}
	result := map[string]bool{}
	for _, point := range series[0].GetPoints() {
		if len(point) != 2 {
			return nil, fmt.Errorf("Expected only two entries in a point for query %q. Got %v", query, point)
		}
		name, ok := point[1].(string)
		if !ok {
			return nil, fmt.Errorf("expected %v to be a string, but it is %T", point[1], point[1])
		}
		result[name] = false
	}
	return result, nil
}

func expectedItemsExist(expectedItems []string, actualItems map[string]bool) bool {
	if len(actualItems) < len(expectedItems) {
		return false
	}
	for _, item := range expectedItems {
		if _, found := actualItems[item]; !found {
			return false
		}
	}
	return true
}

func validateEvents(influxdbClient *influxdb.Client) bool {
	events, err := getInfluxdbData(influxdbClient, eventsQuery)
	if err != nil {
		// We don't fail the test here because the influxdb service might still not be running.
		glog.Errorf("failed to query list of events from influxdb. Query: %q, Err: %v", eventsQuery, err)
		return false
	}
	if len(events) <= 0 {
		glog.Errorf("Query (%q) returned no events. Expected at least 1 event.", eventsQuery)
		return false
	}
	return true
}

func validatePodsAndNodes(influxdbClient *influxdb.Client, expectedPods, expectedNodes []string) bool {
	pods, err := getInfluxdbData(influxdbClient, podlistQuery)
	if err != nil {
		// We don't fail the test here because the influxdb service might still not be running.
		glog.Errorf("failed to query list of pods from influxdb. Query: %q, Err: %v", podlistQuery, err)
		return false
	}
	nodes, err := getInfluxdbData(influxdbClient, nodelistQuery)
	if err != nil {
		glog.Errorf("failed to query list of nodes from influxdb. Query: %q, Err: %v", nodelistQuery, err)
		return false
	}
	if !expectedItemsExist(expectedPods, pods) {
		glog.Errorf("failed to find all expected Pods.\nExpected: %v\nActual: %v", expectedPods, pods)
		return false
	}
	if !expectedItemsExist(expectedNodes, nodes) {
		glog.Errorf("failed to find all expected Nodes.\nExpected: %v\nActual: %v", expectedNodes, nodes)
		return false
	}
	return true
}

func runTest(kubeVersion string, t *testing.T) {
	fm, err := newKubeFramework(t, kubeVersion)
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
	expectedNodes, err := fm.GetNodes()
	require.NoError(t, err, "failed to get list of nodes")
	expectedPods, err := fm.GetPods()
	require.NoError(t, err, "failed to get list of pods")

	startTime := time.Now()
	success := false
	for {
		if validatePodsAndNodes(influxdbClient, expectedPods, expectedNodes) && validateEvents(influxdbClient) {
			success = true
			break
		}
		if time.Since(startTime) >= testTimeout {
			break
		}
		time.Sleep(sleepBetweenAttempts)
	}
	require.True(t, success, "HeapsterInfluxDB test failed with kube version %q", kubeVersion)
	glog.Infof("HeapsterInfluxDB test passed with kube version %q", kubeVersion)
	deleteAll(fm, *namespace, services, replicationControllers)

}

func TestHeapsterInfluxDBWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kubernetes integration test.")
	}
	tempDir, err := ioutil.TempDir("", "deploy")
	require.NoError(t, err, "failed to create temporary directory")
	defer os.RemoveAll(tempDir)

	if *kubeVersions == "" {
		runTest("", t)
		return
	}
	kubeVersionsList := strings.Split(*kubeVersions, ",")
	for _, kubeVersion := range kubeVersionsList {
		runTest(kubeVersion, t)
	}
}
