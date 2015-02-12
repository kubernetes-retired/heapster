Heapster
===========

Heapster enables monitoring of Clusters using [cAdvisor](https://github.com/google/cadvisor).

Heapster supports [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) natively and collects resource usage of all the Pods running in the cluster. It was built to showcase the power of core Kubernetes concepts like labels and pods and the awesomeness that is cAdvisor. 

Heapster can be used to enable cluster wide monitoring on other Cluster management solutions by running a simple cluster specific buddy container that will help heapster with discovery of hosts. For example, take a look at [this guide](clusters/coreos/README.md) for setting up Cluster monitoring in [CoreOS](https://coreos.com).

#####Run Heapster in a Kubernetes cluster with an Influxdb backend and [Grafana](http://grafana.org/docs/features/influxdb)

_Warning: Virtual Machines need to have at least 2 cores for InfluxDB to perform optimally._

**Setup Kube cluster**

 [Bring up a Kubernetes cluster](https://github.com/GoogleCloudPlatform/kubernetes), if you haven't already. Make sure kubecfg.sh is exported. By default, [cAdvisor](https://github.com/google/cadvisor) runs as a Pod on all nodes using a [static manifest file](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/cluster/saltbase/salt/cadvisor/cadvisor.manifest#L1) that is distributed via salt. Make sure that it is running on port 4194 on all nodes.

**Start all the Pods and Services**

```shell
$ kubectl.sh create -f deploy/
```

Grafana will be accessible at `https://<masterIP>/api/v1beta1/proxy/services/monitoring-grafana/`. Use the master auth to access Grafana.

#####Troubleshooting guide

1. If the Grafana service is not accessible, chances are it might not be running. Use `kubectl.sh` to verify that `heapster`, and `influxdb & grafana` Pods are alive.
```shell
$ kubectl.sh get pods
```
```shell
$ kubectl.sh get services
```
2. If the default Grafana dashboard doesn't show any graphs, check the heapster logs. `kubectl.sh log <heapster pod name>`. Look for any errors related to accessing the kubernetes master or the kubelet.
3. To access the InfluxDB UI, you will have to open up the InfluxDB UI port(8083) on the nodes. Find out the IP of the Node where InfluxDB is running to access the UI - `http://<NodeIP>:8083/`
```shell
gcloud compute firewall-rules create monitoring-heapster --allow "tcp:8083" --target-tags=kubernetes-minion
```
Note: We are working on exposing the InfluxDB UI using the proxy service on the Kubernetes master.
4. If you find InfluxDB to be using up a lot of CPU or Memory, consider placing Resource Restrictions on the InfluxDB+Grafana Pod. You can add `cpu: <millicores>` and `memory: <bytes>` in the [Controller Spec](deploy/influxdb-grafana-controller.yaml) and relaunch the Controller.
```shell
deploy/kube.sh restart
```

#####Hints
* To enable memory and swap accounting on the minions follow the instructions [here](https://docs.docker.com/installation/ubuntulinux/#memory-and-swap-accounting)

#####How heapster works on Kubernetes:
1. Discovers all minions in a Kubernetes cluster
2. Collects container statistics from the kubelets running on the minions
2. Organizes stats into Pods and adds kubernetes specific metadata.
3. Stores Pod stats in a configurable backend

Along with each container stat entry, it's Pod ID, Container name, Pod IP, Hostname and Labels are also stored. Labels are stored as key:value pairs.

Heapster currently supports [InfluxDB](http://influxdb.com), and BigQuery backends. Patches are welcome for adding more storage backends.

#### Community

Contributions, questions, and comments are all welcomed and encouraged! Heapster and cAdvisor developers hang out in [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  We also have the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
