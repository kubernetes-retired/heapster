# Run Heapster in a Kubernetes cluster with an InfluxDB backend and a Grafana UI

_Warning: Virtual machines need to have at least 2 cores for InfluxDB to perform optimally._

### Setup a Kubernetes cluster
[Bring up a Kubernetes cluster](https://github.com/GoogleCloudPlatform/kubernetes), if you haven't already. Ensure that `kubecfg.sh` is exported. By default, [cAdvisor](https://github.com/google/cadvisor) runs on all nodes on port 4194.

### Start all of the pods and services
```shell
$ kubectl.sh create -f deploy/kube-config/influxdb/
```

Grafana will be accessible at `https://<masterIP>/api/v1beta1/proxy/services/monitoring-grafana/`. Use the master auth to access Grafana.

## Troubleshooting guide
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

4. If you find InfluxDB to be using up a lot of CPU or memory, consider placing resource restrictions on the `InfluxDB & Grafana` pod. You can add `cpu: <millicores>` and `memory: <bytes>` in the [Controller Spec](deploy/kube-config/influxdb/influxdb-grafana-controller.yaml) and relaunch the controllers:

	```shell
$ deploy/kube.sh restart
	```

## Hints
* To enable memory and swap accounting on the minions, follow [these instructions](https://docs.docker.com/installation/ubuntulinux/#memory-and-swap-accounting).
