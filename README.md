# Heapster
Heapster enables monitoring of clusters using [cAdvisor](https://github.com/google/cadvisor).

Heapster supports [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) natively and collects resource usage of all the pods running in the cluster. It was built to showcase the power of core Kubernetes concepts like labels and pods and the awesomeness that is cAdvisor.

Heapster can be used to enable cluster-wide monitoring on other cluster management solutions by running a simple cluster-specific buddy container that will help Heapster with discovery of hosts. For example, take a look at [this guide](clusters/coreos/README.md) for setting up cluster monitoring in [CoreOS](https://coreos.com) without using Kubernetes.

#### Run Heapster in a Kubernetes cluster with an Influxdb backend and [Grafana](http://grafana.org/docs/features/influxdb)

_Warning: Virtual machines need to have at least 2 cores for InfluxDB to perform optimally._

##### Setup a Kubernetes cluster
[Bring up a Kubernetes cluster](https://github.com/GoogleCloudPlatform/kubernetes), if you haven't already. Ensure that `kubecfg.sh` is exported. By default, [cAdvisor](https://github.com/google/cadvisor) runs as a pod on all nodes using a [static manifest file](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/cluster/saltbase/salt/cadvisor/cadvisor.manifest#L1) that is distributed via salt. Make sure that it is running on port 4194 on all nodes.

##### Start all of the pods and services
```shell
$ kubectl.sh create -f deploy/
```

Grafana will be accessible at `https://<masterIP>/api/v1beta1/proxy/services/monitoring-grafana/`. Use the master auth to access Grafana.

##### Troubleshooting guide
1. If the Grafana service is not accessible, chances are it might not be running. Use `kubectl.sh` to verify that the `heapster` and `influxdb & grafana` pods are alive.

	```shell
$ kubectl.sh get pods
	```

	```shell
$ kubectl.sh get services
	```

2. If the default Grafana dashboard doesn't show any graphs, check the Heapster logs with `kubectl.sh log <heapster pod name>`. Look for any errors related to accessing the Kubernetes master or the kubelet.

3. To access the InfluxDB UI, you will have to open up the InfluxDB UI port (8083) on the nodes. You can do so by creating a firewall rule:

	```shell
$ gcloud compute firewall-rules create monitoring-heapster --allow "tcp:8083" "tcp:8086" --target-tags=kubernetes-minion
	```

	Then, find out the IP address of the node where InfluxDB is running and point your web browser to `http://<nodeIP>:8083/`.

	_Note: We are working on exposing the InfluxDB UI using the proxy service on the Kubernetes master._

4. If you find InfluxDB to be using up a lot of CPU or memory, consider placing resource restrictions on the `InfluxDB & Grafana` pod. You can add `cpu: <millicores>` and `memory: <bytes>` in the [Controller Spec](deploy/influxdb-grafana-controller.yaml) and relaunch the controllers:

	```shell
$ deploy/kube.sh restart
	```

##### Hints
* To enable memory and swap accounting on the minions, follow [these instructions](https://docs.docker.com/installation/ubuntulinux/#memory-and-swap-accounting).

* If the Grafana UI is not accessible via the proxy on the master (URL mentioned above), open up port 80 on the nodes, find out the node on which Grafana container is running, and access the Grafana UI at `http://<minion-ip>/`

	```shell
gcloud compute firewall-rules update monitoring-heapster --allow "tcp:80" --target-tags=kubernetes-minion
echo "Grafana URL: http://$(kubectl.sh get pods -l name=influxGrafana -o yaml | grep hostIP | awk '{print $2}')/"
	```

#### How Heapster works on Kubernetes
1. Discovers all minions in a Kubernetes cluster
2. Collects container statistics from the kubelets running on the nodes
2. Organizes stats into pods and adds Kubernetes-specific metadata
3. Stores pod stats in a configurable backend

Along with each container stat entry, its pod ID, container name, pod IP, hostname and labels are also stored. Labels are stored as `key:value` pairs.

Heapster currently supports [InfluxDB](http://influxdb.com) and [BigQuery](https://cloud.google.com/bigquery/) backends. We welcome patches that add additional storage backends.

#### Community
Contributions, questions, and comments are all welcomed and encouraged! Heapster and cAdvisor developers hang out in the [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  You can also reach us on the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
