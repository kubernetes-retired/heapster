package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

type KafkaCluster struct {
	// cluster size == number of kafka nodes
	size           int
	kafkaDockerDir string
	verbose        bool

	mu         sync.Mutex
	containers []*Container
}

type Container struct {
	cluster *KafkaCluster

	ID     string `json:"Id"`
	Image  string
	Args   []string
	Config struct {
		Cmd          []string
		Env          []string
		ExposedPorts map[string]interface{}
	}
	NetworkSettings struct {
		Gateway   string
		IPAddress string
		Ports     map[string][]PortMapping
	}
}

type PortMapping struct {
	HostIP   string `json:"HostIp"`
	HostPort string
}

func NewKafkaCluster(kafkaDockerDir string, size int) *KafkaCluster {
	if size < 4 {
		fmt.Fprintln(os.Stderr,
			"WARNING: creating cluster smaller than 4 nodes is not sufficient for all topics")
	}
	return &KafkaCluster{
		kafkaDockerDir: kafkaDockerDir,
		size:           size,
		verbose:        testing.Verbose(),
	}
}

// RunningKafka returns true if container is running kafka node
func (c *Container) RunningKafka() bool {
	return c.Args[1] == "start-kafka.sh"
}

// Start starts current container
func (c *Container) Start() error {
	return c.cluster.ContainerStart(c.ID)
}

// Stop stops current container
func (c *Container) Stop() error {
	return c.cluster.ContainerStop(c.ID)
}

func (c *Container) Kill() error {
	return c.cluster.ContainerKill(c.ID)
}

// Start start zookeeper and kafka nodes using docker-compose command. Upon
// successful process spawn, cluster is scaled to required amount of nodes.
func (cluster *KafkaCluster) Start() error {
	cluster.mu.Lock()
	defer cluster.mu.Unlock()

	// ensure  cluster is not running
	if err := cluster.Stop(); err != nil {
		return fmt.Errorf("cannot ensure stop cluster: %s", err)
	}
	if err := cluster.removeStoppedContainers(); err != nil {
		return fmt.Errorf("cannot cleanup dead containers: %s", err)
	}

	upCmd := cluster.cmd("docker-compose", "up", "-d")
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("docker-compose error: %s", err)
	}

	scaleCmd := cluster.cmd("docker-compose", "scale", fmt.Sprintf("kafka=%d", cluster.size))
	if err := scaleCmd.Run(); err != nil {
		_ = cluster.Stop()
		return fmt.Errorf("cannot scale kafka: %s", err)
	}

	containers, err := cluster.Containers()
	if err != nil {
		_ = cluster.Stop()
		return fmt.Errorf("cannot get containers info: %s", err)
	}
	cluster.containers = containers
	return nil
}

// Containers inspect all containers running within cluster and return
// information about them.
func (cluster *KafkaCluster) Containers() ([]*Container, error) {
	psCmd := cluster.cmd("docker-compose", "ps", "-q")
	var buf bytes.Buffer
	psCmd.Stdout = &buf
	if err := psCmd.Run(); err != nil {
		return nil, fmt.Errorf("cannot list processes: %s", err)
	}

	rd := bufio.NewReader(&buf)
	inspectArgs := []string{"inspect"}
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("cannot read \"ps\" output: %s", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		inspectArgs = append(inspectArgs, line)
	}

	inspectCmd := cluster.cmd("docker", inspectArgs...)
	buf.Reset()
	inspectCmd.Stdout = &buf
	if err := inspectCmd.Run(); err != nil {
		return nil, fmt.Errorf("inspect failed: %s", err)
	}
	var containers []*Container
	if err := json.NewDecoder(&buf).Decode(&containers); err != nil {
		return nil, fmt.Errorf("cannot decode inspection: %s", err)
	}
	for _, c := range containers {
		c.cluster = cluster
	}
	return containers, nil
}

// Stop stop all services running for the cluster by sending SIGINT to
// docker-compose process.
func (cluster *KafkaCluster) Stop() error {
	cmd := cluster.cmd("docker-compose", "stop", "-t", "0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose stop failed: %s", err)
	}
	_ = cluster.removeStoppedContainers()
	return nil
}

// KafkaAddrs return list of kafka node addresses as strings, in form
// <host>:<port>
func (cluster *KafkaCluster) KafkaAddrs() ([]string, error) {
	containers, err := cluster.Containers()
	if err != nil {
		return nil, fmt.Errorf("cannot get containers info: %s", err)
	}
	addrs := make([]string, 0)
	for _, container := range containers {
		ports, ok := container.NetworkSettings.Ports["9092/tcp"]
		if !ok || len(ports) == 0 {
			continue
		}
		addrs = append(addrs, fmt.Sprintf("%s:%s", ports[0].HostIP, ports[0].HostPort))
	}
	return addrs, nil
}

func (cluster *KafkaCluster) ContainerStop(containerID string) error {
	stopCmd := cluster.cmd("docker", "stop", containerID)
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("cannot stop %q container: %s", containerID, err)
	}
	return nil
}

func (cluster *KafkaCluster) ContainerKill(containerID string) error {
	killCmd := cluster.cmd("docker", "kill", containerID)
	if err := killCmd.Run(); err != nil {
		return fmt.Errorf("cannot kill %q container: %s", containerID, err)
	}
	return nil
}

func (cluster *KafkaCluster) ContainerStart(containerID string) error {
	startCmd := cluster.cmd("docker", "start", containerID)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("cannot start %q container: %s", containerID, err)
	}
	return nil
}

func (cluster *KafkaCluster) cmd(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	if cluster.verbose {
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
	}
	c.Dir = cluster.kafkaDockerDir
	return c
}

func (cluster *KafkaCluster) removeStoppedContainers() error {
	rmCmd := cluster.cmd("docker-compose", "rm", "-f")
	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("docker-compose rm error: %s", err)
	}
	return nil
}
