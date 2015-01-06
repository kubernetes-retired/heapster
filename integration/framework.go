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

	"github.com/golang/glog"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
)

type kubeFramework interface {
	// Cleanup cluster
	Cleanup() error

	// The testing.T used by the framework and the current test.
	T() *testing.T

	// Kube client
	Client() *kube_client.Client
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

var workDir = flag.String("work_dir", "/tmp/heapster_test", "Filesystem path where test files will be stored. Files will persist across runs to speed up tests.")

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
                }
        }
        output := strings.Join(lines, "\n")
        err = ioutil.WriteFile(kubeConfigFilePath, []byte(output), 0644)
        if err != nil {
		return err
        }
	return nil
}

func setupNewCluster(kubeBaseDir string) error {
	out, err := exec.Command(path.Join(kubeBaseDir, "cluster", "kube-up.sh")).CombinedOutput()
	if err != nil {
		glog.Infof("failed to bring up cluster. Output: %s\nErr: %q", out, err)
		return fmt.Errorf("failed to bring up cluster - %q", err)
	}
	return nil
}

func tearDownCluster(kubeBaseDir string) error {
	out, err := exec.Command(path.Join(kubeBaseDir, "cluster", "kube-down.sh")).CombinedOutput()
	if err != nil {
		glog.V(1).Infof("failed to tear down cluster. Output: %s\nErr: %q", out, err)
		return fmt.Errorf("failed to tear down cluster - %q", err)
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
			glog.V(1).Infof("Master external IP line %v", externalIPLine)
			return strings.TrimSpace(externalIPLine[1]), nil
		}
	}
	return "", fmt.Errorf("External IP not found for the master. 'gcloud compute instances list kubernetes-master' returned %s", out)
}

func downloadRelease(workDir, version string) error {
	// Temporary download path.
	downloadPath := path.Join(workDir, "kube")
	// Format url.
	downloadUrl := fmt.Sprintf(imageUrlTemplate, version)
	glog.V(1).Infof("About to download kube release using url: %q", downloadUrl)

	// Download kube code and store it in a temp dir.
	if err := exec.Command("wget", downloadUrl, "-O", downloadPath).Run(); err != nil {
		return fmt.Errorf("failed to wget kubernetes release @ %s - %q", downloadUrl, err)
	}

	// Un-tar kube release.
	if err := exec.Command("tar", "-xf", downloadPath, "-C", workDir).Run(); err != nil {
		return fmt.Errorf("failed to un-tar kubernetes release at %s - %q", downloadPath, err)
	}
	return nil
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

	// Teardown any pre-existing cluster - This can happen due to a failed test or bad release.
	if err := tearDownCluster(kubeBaseDir); err != nil {
		return nil, err
	}

	// Setup kube cluster 
	if err := setupNewCluster(kubeBaseDir); err != nil {
		return nil, fmt.Errorf("failed to setup kube cluster - %q", err)	
	}

	glog.V(1).Infof("Successfully setup new kubernetes cluster version %s", version)

	// Setup kube client
	masterIP, err := getMasterIP()
	if err != nil {
		return nil, err
	}
	glog.V(1).Infof("Kubernetes master IP is %s", masterIP)

	kubeClient := kube_client.NewOrDie(&kube_client.Config{
		Host:     "http://" + masterIP,
		Version:  "v1beta1",
		Insecure: true,
	})

	return &realKubeFramework{
		kubeClient: kubeClient,
		t:        t,
		baseDir: kubeBaseDir,
	}, nil	
}

func (self *realKubeFramework) T() *testing.T {
	return self.t
}

func (self *realKubeFramework) Cleanup() error {
	if err := tearDownCluster(self.baseDir); err != nil {
		return err
	}

	return nil
}

func (self *realKubeFramework) Client() *kube_client.Client {
	return self.kubeClient
}
