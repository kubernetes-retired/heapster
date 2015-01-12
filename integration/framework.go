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

	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kube_clientauth "github.com/GoogleCloudPlatform/kubernetes/pkg/clientauth"
	"github.com/golang/glog"
)

type kubeFramework interface {
	// The testing.T used by the framework and the current test.
	T() *testing.T

	// Kube client
	Client() *kube_client.Client

	// Run kubectl.sh command
	RunKubectlCmd(cmd ...string) (string, error)

	// Destroy cluster
	DestroyCluster()
}

type realKubeFramework struct {
	// Kube client.
	kubeClient *kube_client.Client

	// The version of the kube cluster
	version string

	// Master host for this framework
	master string

	baseDir string
	// Testing framework in use
	t *testing.T
}

const imageUrlTemplate = "https://github.com/GoogleCloudPlatform/kubernetes/releases/download/v%s/kubernetes.tar.gz"

var (
	authConfig         = flag.String("auth_config", os.Getenv("HOME")+"/.kubernetes_auth", "Path to the auth info file.")
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
	out, err := runKubeClusterCommand(kubeBaseDir, "kube-up.sh")
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

func getKubeClient() (*kube_client.Client, error) {
	masterIP, err := getMasterIP()
	if err != nil {
		return nil, err
	}
	glog.V(1).Infof("Kubernetes master IP is %s", masterIP)

	config := kube_client.Config{
		Host:    "https://" + masterIP,
		Version: "v1beta1",
	}

	auth, err := kube_clientauth.LoadFromFile(*authConfig)
	if err != nil {
		return nil, fmt.Errorf("error loading auth - %q", err)
	}
	config, err = auth.MergeWithConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating client - %q", err)
	}

	kubeClient, err := kube_client.New(&config)
	if err != nil {
		return nil, fmt.Errorf("error creating client - %q", err)
	}

	return kubeClient, nil
}

func validateCluster(baseDir string) bool {
	out, err := runKubeClusterCommand(baseDir, "validate-cluster.sh")
	if err != nil {
		glog.V(1).Infof("cluster validation failed - %q\n %s", err, out)
		return false
	}
	return true
}

func newKubeFramework(t *testing.T, version string) (kubeFramework, error) {
	// All integration tests are large.
	if testing.Short() {
		t.Skip("Skipping framework test in short mode")
	}

	// Create a temp dir to store the kube release files.
	tempDir := path.Join(*workDir, version)
	if !exists(tempDir) {
		if err := os.MkdirAll(tempDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create a temp dir at %s - %q", tempDir, err)
		}
		glog.V(1).Infof("Successfully setup work dir at %s", tempDir)
	}

	kubeBaseDir := path.Join(tempDir, "kubernetes")

	if !exists(kubeBaseDir) {
		if err := downloadRelease(tempDir, version); err != nil {
			return nil, err
		}
		glog.V(1).Infof("Successfully downloaded kubernetes release at %s", tempDir)
	}

	// Disable monitoring
	if err := disableClusterMonitoring(kubeBaseDir); err != nil {
		return nil, fmt.Errorf("failed to disable cluster monitoring in kube cluster config - %q", err)
	}
	glog.V(1).Info("Disabled cluster monitoring")

	if !*useExistingCluster {
		// TODO(vishh): Do a kube-push if a cluster already exists and bring down monitoring jobs.
		// Teardown any pre-existing cluster - This can happen due to a failed test or bad release.
		// Ignore teardown errors since a cluster may not even exist.
		destroyCluster(kubeBaseDir)

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
				return nil, fmt.Errorf("failed to setup cluster - %q", err)
			}
		}

		glog.V(1).Infof("Successfully setup new kubernetes cluster version %s", version)
	}

	// Setup kube client
	kubeClient, err := getKubeClient()
	if err != nil {
		return nil, err
	}

	return &realKubeFramework{
		kubeClient: kubeClient,
		t:          t,
		baseDir:    kubeBaseDir,
	}, nil
}

func (self *realKubeFramework) T() *testing.T {
	return self.t
}

func (self *realKubeFramework) Client() *kube_client.Client {
	return self.kubeClient
}

func (self *realKubeFramework) RunKubectlCmd(args ...string) (string, error) {
	cmd := exec.Command(path.Join(self.baseDir, "cluster", "kubectl.sh"), args...)
	glog.V(2).Infof("about to run cmd: %+v", cmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (self *realKubeFramework) DestroyCluster() {
	if !*useExistingCluster {
		destroyCluster(self.baseDir)
	}
}
