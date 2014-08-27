package sources

import (
	"flag"
	"fmt"
	"net"
	"strings"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kube_labels "github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/golang/glog"
)

var (
	argMaster     = flag.String("kubernetes_master", "", "Kubernetes master IP")
	argMasterAuth = flag.String("kubernetes_master_auth", "", "username:password to access the master")
)

type KubeMasterSource struct {
	client *kube_client.Client
}

// Returns a map of minion hostnames to their corresponding IPs.
func (self *KubeMasterSource) ListMinions() (map[string]string, error) {
	minions, err := self.client.ListMinions()
	if err != nil {
		return nil, err
	}
	hosts := make(map[string]string, 0)
	for _, value := range minions.Items {
		addrs, err := net.LookupIP(value.ID)
		if err == nil {
			hosts[value.ID] = addrs[0].String()
		} else {
			glog.Errorf("Skipping host %s since looking up its IP failed - %s", value.ID, err)
		}
	}

	return hosts, nil
}

func (self *KubeMasterSource) parsePod(pod *kube_api.Pod) *Pod {
	localPod := Pod{
		Name:       pod.DesiredState.Manifest.ID,
		Hostname:   pod.CurrentState.Host,
		Status:     string(pod.CurrentState.Status),
		PodIP:      pod.CurrentState.PodIP,
		Labels:     make(map[string]string, 0),
		Containers: make([]Container, 0),
	}
	for key, value := range pod.Labels {
		localPod.Labels[key] = value
	}
	for _, container := range pod.DesiredState.Manifest.Containers {
		localContainer := Container{
			Name: container.Name,
			ID:   pod.CurrentState.Info[container.Name].ID,
		}
		localPod.Containers = append(localPod.Containers, localContainer)
	}
	return &localPod
}

// Returns a map of minion hostnames to the Pods running in them.
func (self *KubeMasterSource) ListPods() ([]Pod, error) {
	pods, err := self.client.ListPods(kube_labels.Everything())
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

func NewKubeMasterSource() (*KubeMasterSource, error) {
	if len(*argMaster) == 0 {
		return nil, fmt.Errorf("kubernetes_master flag not specified")
	}
	if len(*argMasterAuth) == 0 || len(strings.Split(*argMasterAuth, ":")) != 2 {
		return nil, fmt.Errorf("kubernetes_master_auth invalid")
	}
	authInfo := strings.Split(*argMasterAuth, ":")
	kubeAuthInfo := kube_client.AuthInfo{authInfo[0], authInfo[1]}
	kubeClient := kube_client.New("https://"+*argMaster, &kubeAuthInfo)
	return &KubeMasterSource{
		client: kubeClient,
	}, nil
}
