# Run Heapster in a Kubernetes cluster with an InfluxDB backend and a Grafana and/or Chronograf UI

### Setup a Kubernetes cluster
[Bring up a Kubernetes cluster](https://github.com/kubernetes/kubernetes), if you haven't already.
Ensure that you are able to interact with the cluster via `kubectl` (this may be `kubectl.sh` if using
the local-up-cluster in the Kubernetes repository).

### Start all of the pods and services

In order to deploy Heapster and InfluxDB, you will need to create the Kubernetes resources
described by the contents of [deploy/kube-config/influxdb](../deploy/kubeconfig/influxdb).
Ensure that you have a valid checkout of Heapster and are in the root directory of
the Heapster repository, and then run

```shell
$ kubectl create -f deploy/kube-config/influxdb/
```

Both Grafana & Chronograf services request a LoadBalancer, by default. If that is not available in your cluster (i.e. `minikube`), consider changing that to NodePort. Use the external IP assigned to the service to access the UI.

**Grafana**

The default user name and password is 'admin'.
Once you login to Grafana, add a datasource that is InfluxDB. The URL for InfluxDB will be `http://localhost:8086`. Database name is 'k8s'. Default user name and password is 'root'.
Grafana documentation for InfluxDB [here](http://docs.grafana.org/datasources/influxdb/).

Grafana is set up to auto-populate nodes and pods using templates.

**Chronograf**

Once you login to Chronograf, add a datasource for InfluxDB.  The host will be `influxdb` and port will be `8086`

**InfluxDB**

Take a look at the [storage schema](storage-schema.md) to understand how metrics are stored in InfluxDB.

Grafana is set up to auto-populate nodes and pods using templates.

## Troubleshooting guide
1. If the Grafana service is not accessible, chances are it might not be running. Use `kubectl.sh` to verify that the `heapster` and `influxdb & grafana` pods are alive.

	kubectl --namespace kube-system get pods

	kubectl --namespace kube-system get services

2. To access the InfluxDB UI, you will have to make the InfluxDB service externally visible, similar to how Grafana & Chronograf are made publicly accessible.

3. If you find InfluxDB to be using up a lot of CPU or memory, consider placing resource restrictions on the `InfluxDB & Grafana` pod. You can add `cpu: <millicores>` and `memory: <bytes>` in the [Controller Spec](../deploy/kube-config/influxdb/influxdb-grafana-controller.yaml) and relaunch the controllers:
