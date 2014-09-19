Heapster
===========

_Warning: Heapster currently only works on Google Compute Engine!_

Heapster enables monitoring of Clusters using [cAdvisor](https://github.com/google/cadvisor). It is a *demo app* that demonstrates one possible way of monitoring various types of Clusters using cAdvisor.

Heapster supports [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) natively and collects resource usage of all the Pods running in the cluster. It was built to showcase the power of core Kubernetes concepts like labels and pods and the awesomeness that is cAdvisor. 

Heapster can be used to enable cluster wide monitoring on other Cluster management solutions by running a simple cluster specific buddy container that will help heapster with discovery of hosts. For example, take a look at [this guide](clusters/coreos/README.md) for setting up Cluster monitoring in [CoreOS](https://coreos.com).


#####How heapster works on Kubernetes:
1. Discovers all minions in a Kubernetes cluster
2. Collects container statistics from the cadvisors running on the minions
2. Organizes stats into Pods
3. Stores Pod stats in a configurable backend

Along with each container stat entry, it's Pod ID, Container name, Pod IP, Hostname and Labels are also stored. Labels are stored as key:value pairs.

Heapster currently supports in-memory and [InfluxDB](http://influxdb.com) backends. Patches are welcome for adding more storage backends.

#####Run Heapster in a Kubernetes cluster with an Influxdb backend and [Grafana](http://grafana.org/docs/features/influxdb)

**Step 1: Setup Kube cluster**

Fork the Kubernetes repository and [turn up a Kubernetes cluster](https://github.com/GoogleCloudPlatform/kubernetes-new#contents), if you haven't already. Make sure kubecfg.sh is exported.

**Step 2: Start a Pod with Influxdb, grafana and elasticsearch**

```shell
$ kubecfg.sh -c influx-grafana/deploy/grafana-influxdb-pod.json create pods
```

**Step 3: Start Influxdb service**

```shell
$ kubecfg.sh -c influx-grafana/deploy/grafana-influxdb-service.json create services
```

**Step 4: Update firewall rules**

Open up ports tcp:80,8083,8086,9200.
```shell
$ gcutil addfirewall --allowed=tcp:80,tcp:8083,tcp:8086,tcp:9200 --target_tags=kubernetes-minion heapster
```

**Step 5: Configure cluster information for heapster Pod**

Open deploy/heapster-pod.json and update the following environment variables:
* Set 'KUBE_MASTER' to the internal IP address of the master - `$ gcutil listinstances | grep kubernetes-master | awk '{print $8}'`
* Set 'KUBE_MASTER_AUTH' to be the the username and password of the master. The format is username:password. - `$ cat ~/.kubernetes_auth`

**Step 6: Start Heapster Pod**

```shell
$ kubecfg.sh -c deploy/heapster-pod.json create pods
```

Verify that all the pods and services are up and running:

```shell
$ kubecfg.sh list pods
```
```shell
$ kubecfg.sh list services
```

To start monitoring the cluster using grafana, find out the the external IP of the minion where the 'influx-grafana' Pod is running from the [Google Cloud Console][cloud-console] or the `gcutil` tool, and visit `http://<minion-ip>:80`. 

To access the Influxdb UI visit  `http://<minion-ip>:8083`.


```shell
$ gcutil listinstances
```

#####Hints
* Grafana's default username and password is 'admin'. You can change that by modifying the grafana container [here](influx-grafana/deploy/grafana-influxdb-pod.json)
* To enable memory and swap accounting on the minions follow the instructions [here](https://docs.docker.com/installation/ubuntulinux/#memory-and-swap-accounting)

#### Community

Contributions, questions, and comments are all welcomed and encouraged! Heapster and cAdvisor developers hang out in [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  We also have the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).

[cloud-console]: https://console.developer.google.com
[gcutil-docs]: https://developers.google.com/compute/docs/gcutil/reference/firewall#addfirewall
