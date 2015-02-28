# Heapster
Heapster enables monitoring of clusters using [cAdvisor](https://github.com/google/cadvisor).

Heapster supports [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) natively and collects resource usage of all the pods running in the cluster. It was built to showcase the power of core Kubernetes concepts like labels and pods and the awesomeness that is cAdvisor.

### Running Heapster on Kubernetes

To run Heapster on a Kubernetes cluster targetting [InfluxDB](http://influxdb.com) and [Grafana](http://grafana.org/docs/features/influxdb) use [this guide](docs/influxdb.md). Alternatively you can also use [Google Cloud Monitoring](https://cloud.google.com/monitoring/) with [this guide](docs/gcm.md).

While running on Kubernetes, Heapster does the following:

1. Discovers all minions in a Kubernetes cluster
2. Collects container statistics from the kubelets running on the nodes
3. Organizes stats into pods and adds Kubernetes-specific metadata
4. Stores pod stats in a configurable backend

Along with each container stat entry, its pod ID, container name, pod IP, hostname and labels are also stored. Labels are stored as `key:value` pairs.

Heapster currently supports [InfluxDB](http://influxdb.com), [Google Cloud Monitoring](https://cloud.google.com/monitoring/), and [BigQuery](https://cloud.google.com/bigquery/) backends. We welcome patches that add additional storage backends.

### Running outside of Kubernetes

Heapster can be used to enable cluster-wide monitoring on other cluster management solutions by running a simple cluster-specific buddy container that will help Heapster with discovery of hosts. For example, take a look at [this guide](clusters/coreos/README.md) for setting up cluster monitoring in [CoreOS](https://coreos.com) without using Kubernetes.

It is also possible to run Heapster standalone on a host with cAdvisor using [this guide](docs/standalone.md).

### Community
Contributions, questions, and comments are all welcomed and encouraged! Heapster and cAdvisor developers hang out in the [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  You can also reach us on the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
