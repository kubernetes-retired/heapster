# Heapster
Heapster enables Container Cluster Monitoring. 

Internally, heapster uses [cAdvisor](https://github.com/google/cadvisor) for compute resource usage metrics.

Heapster currently supports [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) and CoreOS natively. It can be extended to support other cluster management solutions easily.
While running in a Kube cluster, heapster collects compute resource usage of all pods and nodes.

Source configuration is documented [here](docs/source-configuration.md).

### Running Heapster on Kubernetes
Heapster supports a pluggable storage backend. It supports [InfluxDB](http://influxdb.com) with [Grafana](http://grafana.org/docs/features/influxdb) and [Google Cloud Monitoring](https://cloud.google.com/monitoring/). We welcome patches that add additional storage backends.

To run Heapster on a Kubernetes cluster with,
- InfluxDB use [this guide](docs/influxdb.md). 
- Google Cloud Monitoring use [this guide](docs/gcm.md).

Take a look at the storage schema [here](docs/storage-schema.md).

### Running Heapster on CoreOS
Heapster communicates with the local fleet server to get cluster information. It expected cAdvisor to be running on all the nodes. Refer to [this guide](docs/coreos.md).

### Running in other cluster management systems.

Heapster can be used to enable cluster-wide monitoring on other cluster management solutions by running a simple cluster-specific buddy container that will help Heapster with discovery of hosts.

### Running in standalone mode.

It is also possible to run Heapster standalone on a host with cAdvisor using [this guide](docs/standalone.md).

#### Troubleshooting guide [here](docs/debugging.md)

### Community
Contributions, questions, and comments are all welcomed and encouraged! Heapster and cAdvisor developers hang out in the [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  You can also reach us on the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
