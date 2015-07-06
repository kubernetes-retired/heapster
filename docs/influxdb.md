# Run Heapster in a Kubernetes cluster with an InfluxDB backend and a Grafana UI

_Warning: Virtual machines need to have at least 2 cores for InfluxDB to perform optimally._

### Setup a Kubernetes cluster
[Bring up a Kubernetes cluster](https://github.com/GoogleCloudPlatform/kubernetes), if you haven't already. Ensure that `kubecfg.sh` is exported.

### Start all of the pods and services
```shell
$ kubectl.sh create -f deploy/kube-config/influxdb/
```

Grafana will be accessible at `https://<masterIP>/api/v1/proxy/namespaces/default/services/monitoring-grafana/`. Use the master auth to access Grafana.

## Production setup for InfluxDB and Grafana

By default, InfluxDB and Grafana are made accessible via a proxy on the Kubernetes apiserver, to make them accessible with minimal configurations on all
Kubernetes deployments.
This is not ideal for production deployments since the proxy will be a bottleneck. 
To use InfluxDB and Grafana in production, we recommend setting up an [External Kubernetes Service](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/docs/services.md#external-services).

#### Guide
1. Remove the environment variable `INFLUXDB_EXTERNAL_URL` from InfluxDB Grafana RC config [here](../deploy/kube-config/influxdb/influxdb-grafana-controller.json).
   This option makes Grafana reach InfluxDB service via the apiserver proxy.

2. Update Grafana service to either include a public IP or setup an external load balancer.
   In the [Grafana service](../deploy/kube-config/influxdb/grafana-service.json),  set `"createExternalLoadBalancer": true` in the Grafana Service Spec or,
   set `"PublicIPs": ["<Externally Accessible IP>"]`.

3. Delete and recreate InfluxDB and Grafana RCs and services.

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

4. If you find InfluxDB to be using up a lot of CPU or memory, consider placing resource restrictions on the `InfluxDB & Grafana` pod. You can add `cpu: <millicores>` and `memory: <bytes>` in the [Controller Spec](../deploy/kube-config/influxdb/influxdb-grafana-controller.json) and relaunch the controllers:

	```shell
$ deploy/kube.sh restart
	```
