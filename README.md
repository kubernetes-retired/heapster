Heapster
===========

Heapster enables monitoring of Clusters using [cAdvisor](https://github.com/google/cadvisor).

Heapster supports [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) natively and collects resource usage of all the Pods running in the cluster. It was built to showcase the power of core Kubernetes concepts like labels and pods and the awesomeness that is cAdvisor. 

Heapster can be used to enable cluster wide monitoring on other Cluster management solutions by running a simple cluster specific buddy container that will help heapster with discovery of hosts. For example, take a look at [this guide](clusters/coreos/README.md) for setting up Cluster monitoring in [CoreOS](https://coreos.com).

#####Run Heapster in a Kubernetes cluster with an Influxdb backend and [Grafana](http://grafana.org/docs/features/influxdb)

**Step 1: Setup Kube cluster**

Fork the Kubernetes repository and [turn up a Kubernetes cluster](https://github.com/GoogleCloudPlatform/kubernetes-new#contents), if you haven't already. Make sure kubecfg.sh is exported. By default, [cAdvisor](https://github.com/google/cadvisor) runs as a Pod on all nodes using a [static manifest file](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/cluster/saltbase/salt/cadvisor/cadvisor.manifest#L1) that is distributed via salt. Make sure that it is running on port 4194 on all nodes.

**Step 2: Start a Pod with Influxdb, grafana and elasticsearch**

```shell
$ kubectl.sh create -f deploy/influxdb-grafana-controller.js
```

**Step 3: Start Influxdb service**

```shell
$ kubectl.sh create -f deploy/influxdb-service.json
```

```shell
$ gcloud compute firewall-rules create monitoring-heapster --allow "tcp:80" "tcp:8083" "tcp:8086" --target-tags=kubernetes-minion
```

**Step 5: Start Heapster Pod**

```shell
$ kubectl.sh create -f deploy/heapster-controller.js
```

Verify that all the pods and services are up and running:

```shell
$ kubectl.sh get pods
```
```shell
$ kubectl.sh get services
```

Grafana will be accessible at `https://<masterIP>/api/v1beta1/proxy/services/monitoring-grafana/`. Use the master auth to access Grafana.

_Warning: Virtual Machines need to have at least 2 cores for InfluxDB to perform optimally._

#####Hints
* To enable memory and swap accounting on the minions follow the instructions [here](https://docs.docker.com/installation/ubuntulinux/#memory-and-swap-accounting)

#####How heapster works on Kubernetes:
1. Discovers all minions in a Kubernetes cluster
2. Collects container statistics from the kubelets running on the minions
2. Organizes stats into Pods
3. Stores Pod stats in a configurable backend

Along with each container stat entry, it's Pod ID, Container name, Pod IP, Hostname and Labels are also stored. Labels are stored as key:value pairs.

Heapster currently supports in-memory and [InfluxDB](http://influxdb.com) backends. Patches are welcome for adding more storage backends.

#### Community

Contributions, questions, and comments are all welcomed and encouraged! Heapster and cAdvisor developers hang out in [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  We also have the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
