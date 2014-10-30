package sources

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kube_labels "github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/golang/glog"
	cadvisor "github.com/google/cadvisor/info"
)


// Kubernetes released supported and tested against.
var kubeVersions = []string{"v0.4.2"}

// Cadvisor port in kubernetes.
const cadvisorPort = 4194

type KubeSource struct {
	client      *kube_client.Client
	lastQuery   time.Time
	kubeletPort string
}

type nodeList CadvisorHosts

// Returns a map of minion hostnames to their corresponding IPs.
func (self *KubeSource) listMinions() (*nodeList, error) {
	nodeList := &nodeList{
		Port:  cadvisorPort,
		Hosts: make(map[string]string, 0),
	}
	minions, err := self.client.Minions().List()
	if err != nil {
		return nil, err
	}
	for _, minion := range minions.Items {
		addrs, err := net.LookupIP(minion.Name)
		if err == nil {
			nodeList.Hosts[minion.Name] = addrs[0].String()
		} else {
			glog.Errorf("Skipping host %s since looking up its IP failed - %s", minion.Name, err)
		}
	}

	return nodeList, nil
}

func (self *KubeSource) parsePod(pod *kube_api.Pod) *Pod {
	localPod := Pod{
		Name:       pod.DesiredState.Manifest.ID,
		ID:         pod.DesiredState.Manifest.UUID,
		Hostname:   pod.CurrentState.Host,
		Status:     string(pod.CurrentState.Status),
		PodIP:      pod.CurrentState.PodIP,
		Labels:     make(map[string]string, 0),
		Containers: make([]*Container, 0),
	}
	for key, value := range pod.Labels {
		localPod.Labels[key] = value
	}
	for _, container := range pod.DesiredState.Manifest.Containers {
		localContainer := newContainer()
		localContainer.Name = container.Name
		localPod.Containers = append(localPod.Containers, localContainer)
	}

	return &localPod
}

// Returns a map of minion hostnames to the Pods running in them.
func (self *KubeSource) getPods() ([]Pod, error) {
	pods, err := self.client.Pods(kube_api.NamespaceDefault).List(kube_labels.Everything())
	if err != nil {
		return nil, err
	}
	out := make([]Pod, 0)
	for _, pod := range pods.Items {
		pod := self.parsePod(&pod)
		out = append(out, *pod)
	}

	return out, nil
}

func (self *KubeSource) getStatsFromKubelet(hostIP, podName, podID, containerName string) (*cadvisor.ContainerSpec, []*cadvisor.ContainerStats, error) {
	var containerInfo cadvisor.ContainerInfo
	values := url.Values{}
	values.Add("num_stats", strconv.Itoa(int(time.Since(self.lastQuery)/time.Second)))
	url := "http://" + hostIP + ":" + self.kubeletPort + filepath.Join("/stats", podName, containerName) + "?" + values.Encode()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	err = PostRequestAndGetValue(&http.Client{}, req, &containerInfo)
	if err != nil {
		glog.Errorf("failed to get stats from kubelet on host with ip %s - %s\n", hostIP, err)
		return nil, nil, nil
	}

	return &containerInfo.Spec, containerInfo.Stats, nil
}

func (self *KubeSource) GetAllStats() (StatsData, error) {
	pods, err := self.getPods()
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		addrs, err := net.LookupIP(pod.Hostname)
		if err != nil {
			glog.Errorf("Skipping host %s since looking up its IP failed - %s", pod.Hostname, err)
			continue
		}
		hostIP := addrs[0].String()
		for _, container := range pod.Containers {
			spec, stats, err := self.getStatsFromKubelet(hostIP, pod.Name, pod.ID, container.Name)
			if err != nil {
				return nil, err
			}
			container.Stats = stats
			container.Spec = *spec
		}
	}
	self.lastQuery = time.Now()

	return pods, nil
}

func newKubeSource() (*KubeSource, error) {
	if len(*argMaster) == 0 {
		return nil, fmt.Errorf("kubernetes_master flag not specified")
	}
	if len(*argMasterAuth) == 0 || len(strings.Split(*argMasterAuth, ":")) != 2 {
		return nil, fmt.Errorf("kubernetes_master_auth invalid")
	}
	authInfo := strings.Split(*argMasterAuth, ":")
	kubeClient := kube_client.NewOrDie(&kube_client.Config{
		Host:     "https://" + *argMaster,
		Version:  "v1beta1",
		Username: authInfo[0],
		Password: authInfo[1],
		Insecure: true,
	})
	glog.Infof("Following Kubernetes versions are supported: %v", kubeVersions)
	return &KubeSource{
		client:      kubeClient,
		lastQuery:   time.Now(),
		kubeletPort: *argKubeletPort,
	}, nil
}
