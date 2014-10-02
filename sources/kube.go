package sources

import (
	"fmt"
	"net"
	"strings"

	kube_api "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kube_client "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kube_labels "github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/golang/glog"
)

const cadvisorPort = 4194

type KubeSource struct {
	client   *kube_client.Client
	cadvisor *cadvisorSource
}

// Returns a map of minion hostnames to their corresponding IPs.
func (self *KubeSource) listMinions() (*CadvisorHosts, error) {
	cadvisorHosts := &CadvisorHosts{
		Port:  cadvisorPort,
		Hosts: make(map[string]string, 0),
	}
	minions, err := self.client.ListMinions()
	if err != nil {
		return nil, err
	}
	for _, value := range minions.Items {
		addrs, err := net.LookupIP(value.ID)
		if err == nil {
			cadvisorHosts.Hosts[value.ID] = addrs[0].String()
		} else {
			glog.Errorf("Skipping host %s since looking up its IP failed - %s", value.ID, err)
		}
	}

	return cadvisorHosts, nil
}

func (self *KubeSource) parsePod(pod *kube_api.Pod) *Pod {
	localPod := Pod{
		Name:       pod.DesiredState.Manifest.ID,
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
		localContainer.ID = pod.CurrentState.Info[container.Name].DetailInfo.ID
		localPod.Containers = append(localPod.Containers, localContainer)
	}
	return &localPod
}

// Returns a map of minion hostnames to the Pods running in them.
func (self *KubeSource) GetPods() ([]Pod, error) {
	pods, err := self.client.ListPods(kube_api.NewContext(), kube_labels.Everything())
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

func (self *KubeSource) GetContainerStats() (HostnameContainersMap, error) {
	hosts, err := self.listMinions()
	if err != nil {
		return nil, err
	}

	return self.cadvisor.fetchData(hosts)
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
	cadvisorSource := newCadvisorSource()
	return &KubeSource{
		client:   kubeClient,
		cadvisor: cadvisorSource,
	}, nil
}
