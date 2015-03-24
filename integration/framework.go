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
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/latest"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/golang/glog"
)

type kubeFramework interface {
	// The testing.T used by the framework and the current test.
	T() *testing.T

	// Kube client
	Client() *kube_client.Client

	// Parses and Returns a replication Controller object contained in 'filePath'
	ParseRC(filePath string) (*api.ReplicationController, error)

	// Parses and Returns a service object contained in 'filePath'
	ParseService(filePath string) (*api.Service, error)

	// Creates a kube service.
	CreateService(ns string, service *api.Service) error

	// Deletes a kube service.
	DeleteService(ns string, service *api.Service) error

	// Creates a kube replication controller.
	CreateRC(ns string, rc *api.ReplicationController) error

	// Deletes a kube replication controller.
	DeleteRC(ns string, rc *api.ReplicationController) error

	// Destroy cluster
	DestroyCluster()

	// Returns a url that provides access to a kubernetes service via the proxy on the apiserver.
	// This url requires master auth.
	GetProxyUrlForService(serviceName string) string

	// Returns the node hostnames.
	GetNodes() ([]string, error)
}

type realKubeFramework struct {
	// Kube client.
	kubeClient *kube_client.Client

	// The version of the kube cluster
	version string

	// Master IP for this framework
	masterIP string

	// The base directory of current kubernetes release.
	baseDir string

	// Testing framework in use
	t *testing.T
}

const imageUrlTemplate = "https://github.com/GoogleCloudPlatform/kubernetes/releases/download/v%s/kubernetes.tar.gz"

var (
	kubeConfig         = flag.String("kube_config", os.Getenv("HOME")+"/.kube/.kubeconfig", "Path to cluster info file.")
	useExistingCluster = flag.Bool("use_existing_cluster", false, "when true uses an existing kube cluster. A cluster needs to exist.")
	workDir            = flag.String("work_dir", "/tmp/heapster_test", "Filesystem path where test files will be stored. Files will persist across runs to speed up tests.")
)

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

const pathToGCEConfig = "cluster/gce/config-default.sh"

func disableClusterMonitoring(kubeBaseDir string) error {
	kubeConfigFilePath := path.Join(kubeBaseDir, pathToGCEConfig)
	input, err := ioutil.ReadFile(kubeConfigFilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, "ENABLE_CLUSTER_MONITORING") {
			lines[i] = "ENABLE_CLUSTER_MONITORING=false"
		} else if strings.Contains(line, "NUM_MINIONS=") {
			lines[i] = "NUM_MINIONS=2"
		}
	}
	output := strings.Join(lines, "\n")
	return ioutil.WriteFile(kubeConfigFilePath, []byte(output), 0644)
}

func runKubeClusterCommand(kubeBaseDir, command string) ([]byte, error) {
	cmd := exec.Command(path.Join(kubeBaseDir, "cluster", command))
	glog.V(2).Infof("about to run %v", cmd)
	return cmd.CombinedOutput()
}

func setupNewCluster(kubeBaseDir string) error {
	cmd := "kube-up.sh"
	destroyCluster(kubeBaseDir)
	out, err := runKubeClusterCommand(kubeBaseDir, cmd)
	if err != nil {
		glog.Errorf("failed to bring up cluster - %q\n%s", err, out)
		return fmt.Errorf("failed to bring up cluster - %q", err)
	}

	return nil
}

func destroyCluster(kubeBaseDir string) error {
	glog.V(1).Info("Bringing down any existing kube cluster")
	out, err := runKubeClusterCommand(kubeBaseDir, "kube-down.sh")
	if err != nil {
		glog.Errorf("failed to tear down cluster - %q\n%s", err, out)
		return fmt.Errorf("failed to tear down kube cluster - %q", err)
	}

	return nil
}

func getMasterIP() (string, error) {
	out, err := exec.Command("gcloud", "compute", "instances", "list", "kubernetes-master", "--format=yaml").CombinedOutput()
	if err != nil {
		return "", err
	}
	glog.V(2).Infof("Output of gcloud compute instance list kubernetes-master:\n %s", out)
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "natIP") {
			externalIPLine := strings.Split(line, ":")
			return strings.TrimSpace(externalIPLine[1]), nil
		}
	}

	return "", fmt.Errorf("external IP not found for the master. 'gcloud compute instances list kubernetes-master' returned %s", out)
}

func downloadRelease(workDir, version string) error {
	// Temporary download path.
	downloadPath := path.Join(workDir, "kube")
	// Format url.
	downloadUrl := fmt.Sprintf(imageUrlTemplate, version)
	glog.V(1).Infof("About to download kube release using url: %q", downloadUrl)

	// Download kube code and store it in a temp dir.
	if err := exec.Command("wget", downloadUrl, "-O", downloadPath).Run(); err != nil {
		return fmt.Errorf("failed to wget kubernetes release @ %q - %v", downloadUrl, err)
	}

	// Un-tar kube release.
	if err := exec.Command("tar", "-xf", downloadPath, "-C", workDir).Run(); err != nil {
		return fmt.Errorf("failed to un-tar kubernetes release at %q - %v", downloadPath, err)
	}
	return nil
}

func getKubeClient() (string, *kube_client.Client, error) {
	masterIP, err := getMasterIP()
	if err != nil {
		return "", nil, err
	}
	glog.V(1).Infof("Kubernetes master IP is %s", masterIP)

	c, err := clientcmd.LoadFromFile(*kubeConfig)
	if err != nil {
		return "", nil, fmt.Errorf("error loading kubeConfig: %v", err.Error())
	}
	config, err := clientcmd.NewDefaultClientConfig(*c, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return "", nil, fmt.Errorf("error parsing kubeConfig: %v", err.Error())
	}

	kubeClient, err := kube_client.New(config)
	if err != nil {
		return "", nil, fmt.Errorf("error creating client - %q", err)
	}

	return masterIP, kubeClient, nil
}

func validateCluster(baseDir string) bool {
	glog.V(1).Info("validating existing cluster")
	out, err := runKubeClusterCommand(baseDir, "validate-cluster.sh")
	if err != nil {
		glog.V(1).Infof("cluster validation failed - %q\n %s", err, out)
		return false
	}
	return true
}

func requireNewCluster(baseDir, version string) bool {
	// Setup kube client
	_, kubeClient, err := getKubeClient()
	if err != nil {
		glog.V(1).Infof("kube client creation failed - %q", err)
		return true
	}
	glog.V(1).Infof("checking if existing cluster can be used")
	versionInfo, err := kubeClient.ServerVersion()
	if err != nil {
		glog.V(1).Infof("failed to get kube version info - %q", err)
		return true
	}
	return !strings.Contains(versionInfo.GitVersion, version)
}

func downloadAndSetupCluster(version string) (baseDir string, err error) {
	// Create a temp dir to store the kube release files.
	tempDir := path.Join(*workDir, version)
	if !exists(tempDir) {
		if err := os.MkdirAll(tempDir, 0700); err != nil {
			return "", fmt.Errorf("failed to create a temp dir at %s - %q", tempDir, err)
		}
		glog.V(1).Infof("Successfully setup work dir at %s", tempDir)
	}

	kubeBaseDir := path.Join(tempDir, "kubernetes")

	if !exists(kubeBaseDir) {
		if err := downloadRelease(tempDir, version); err != nil {
			return "", err
		}
		glog.V(1).Infof("Successfully downloaded kubernetes release at %s", tempDir)
	}

	// Disable monitoring
	if err := disableClusterMonitoring(kubeBaseDir); err != nil {
		return "", fmt.Errorf("failed to disable cluster monitoring in kube cluster config - %q", err)
	}
	glog.V(1).Info("Disabled cluster monitoring")
	if !requireNewCluster(kubeBaseDir, version) {
		glog.V(1).Infof("skipping cluster setup since a cluster with required version already exists")
		return kubeBaseDir, nil
	}

	// Setup kube cluster
	glog.V(1).Infof("Setting up new kubernetes cluster version: %s", version)
	if err := setupNewCluster(kubeBaseDir); err != nil {
		// Cluster setup failed for some reason.
		// Attempting to validate the cluster to see if it failed in the validate phase.
		sleepDuration := 10 * time.Second
		var clusterReady bool = false
		for i := 0; i < int(time.Minute/sleepDuration); i++ {
			if !validateCluster(kubeBaseDir) {
				glog.Infof("Retry validation after %v seconds.", sleepDuration/time.Second)
				time.Sleep(sleepDuration)
			} else {
				clusterReady = true
				break
			}
		}
		if !clusterReady {
			return "", fmt.Errorf("failed to setup cluster - %q", err)
		}
	}
	glog.V(1).Infof("Successfully setup new kubernetes cluster version %s", version)

	return kubeBaseDir, nil
}

func newKubeFramework(t *testing.T, version string) (kubeFramework, error) {
	var err error
	kubeBaseDir := ""
	// All integration tests are large.
	if testing.Short() {
		t.Skip("Skipping framework test in short mode")
	}
	if !*useExistingCluster {
		if version == "" || len(strings.Split(version, ".")) != 3 {
			return nil, fmt.Errorf("invalid kubernetes version specified - %q", version)
		}
		kubeBaseDir, err = downloadAndSetupCluster(version)
		if err != nil {
			return nil, err
		}
	}

	// Setup kube client
	masterIP, kubeClient, err := getKubeClient()
	if err != nil {
		return nil, err
	}

	return &realKubeFramework{
		kubeClient: kubeClient,
		t:          t,
		baseDir:    kubeBaseDir,
		masterIP:   masterIP,
	}, nil
}

func (self *realKubeFramework) T() *testing.T {
	return self.t
}

func (self *realKubeFramework) Client() *kube_client.Client {
	return self.kubeClient
}

func (self *realKubeFramework) loadObject(filePath string) (runtime.Object, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %v", err)
	}
	return latest.Codec.Decode(data)
}

func (self *realKubeFramework) ParseRC(filePath string) (*api.ReplicationController, error) {
	obj, err := self.loadObject(filePath)
	if err != nil {
		return nil, err
	}

	rc, ok := obj.(*api.ReplicationController)
	if !ok {
		return nil, fmt.Errorf("Failed to cast replicationController: %v", obj)
	}
	return rc, nil
}

func (self *realKubeFramework) ParseService(filePath string) (*api.Service, error) {
	obj, err := self.loadObject(filePath)
	if err != nil {
		return nil, err
	}
	service, ok := obj.(*api.Service)
	if !ok {
		return nil, fmt.Errorf("Failed to cast service: %v", obj)
	}
	return service, nil
}

func (self *realKubeFramework) CreateService(ns string, service *api.Service) error {
	_, err := self.kubeClient.Services(ns).Create(service)
	return err
}

func (self *realKubeFramework) DeleteService(ns string, service *api.Service) error {
	if _, err := self.kubeClient.Services(ns).Get(service.Name); err != nil {
		glog.V(1).Infof("cannot find service %q - %v", service.Name, err)
		return nil
	}

	return self.kubeClient.Services(ns).Delete(service.Name)
}

func (self *realKubeFramework) CreateRC(ns string, rc *api.ReplicationController) error {
	_, err := self.kubeClient.ReplicationControllers(ns).Create(rc)
	return err
}

func (self *realKubeFramework) DeleteRC(ns string, inputRc *api.ReplicationController) error {
	rc, err := self.kubeClient.ReplicationControllers(ns).Get(inputRc.Name)
	if err != nil {
		glog.V(1).Infof("Cannot find Replication Controller %q. Skipping deletion - %v", inputRc.Name, err)
		return nil
	}
	rc.Spec.Replicas = 0
	if _, err := self.kubeClient.ReplicationControllers(ns).Update(rc); err != nil {
		return fmt.Errorf("unable to modify replica count for rc %v: %v", inputRc.Name, err)
	}
	if err := self.kubeClient.ReplicationControllers(ns).Delete(rc.Name); err != nil {
		return fmt.Errorf("unable to delete rc %v: %v", inputRc.Name, err)
	}

	return nil
}

func (self *realKubeFramework) DestroyCluster() {
	if !*useExistingCluster {
		destroyCluster(self.baseDir)
	}
}

func (self *realKubeFramework) GetProxyUrlForService(serviceName string) string {
	return fmt.Sprintf("%s/api/v1beta1/proxy/services/%s/", self.masterIP, serviceName)
}

func (self *realKubeFramework) GetNodes() ([]string, error) {
	var nodes []string
	nodeList, err := self.kubeClient.Nodes().List()
	if err != nil {
		return nodes, err
	}

	for _, node := range nodeList.Items {
		name := strings.Split(node.Name, ".")[0]
		nodes = append(nodes, name)
	}
	return nodes, nil
}
