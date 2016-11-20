# Run Heapster in a Kubernetes cluster with an InfluxDB backend and a Grafana UI

### Setup a Kubernetes cluster
[Bring up a Kubernetes cluster](https://github.com/kubernetes/kubernetes), if you haven't already.
Ensure that you are able to interact with the cluster via `kubectl` (this may be `kubectl.sh` if using
the local-up-cluster in the Kubernetes repository).

### Start all of the pods and services

In order to deploy Heapster and InfluxDB, you will need to create the Kubernetes resources
described by the contents of [deploy/kube-config/influxdb](../deploy/kube-config/influxdb).
Ensure that you have a valid checkout of Heapster and are in the root directory of
the Heapster repository, and then run

```shell
$ kubectl create -f deploy/kube-config/influxdb/
```

Grafana service by default requests for a LoadBalancer. If that is not available in your cluster, consider changing that to NodePort. Use the external IP assigned to the Grafana service,
to access Grafana.
The default user name and password is 'admin'.
Once you login to Grafana, add a datasource that is InfluxDB. The URL for InfluxDB will be `http://localhost:8086`. Database name is 'k8s'. Default user name and password is 'root'.
Grafana documentation for InfluxDB [here](http://docs.grafana.org/datasources/influxdb/).

Take a look at the [storage schema](storage-schema.md) to understand how metrics are stored in InfluxDB.

Grafana is set up to auto-populate nodes and pods using templates.

The Grafana web interface can also be accessed via the api-server proxy. The URL should be visible in `kubectl cluster-info` once the above resources are created.

## Troubleshooting guide

See also the [debugging documentation](debugging.md).

1. If the Grafana service is not accessible, it might not be running. Use `kubectl` to verify that the `heapster` and `influxdb & grafana` pods are alive.
    ```
    $ kubectl get pods --namespace=kube-system
    ...
    monitoring-grafana-927606581-0tmdx        1/1       Running   0          6d
    monitoring-influxdb-3276295126-joqo2      1/1       Running   0          15d
    ...
    
    $ kubectl get services --namespace=kube-system monitoring-grafana monitoring-influxdb
    ```

2. To access the InfluxDB UI, you will have to create an InfluxDB service similar to Grafana's. The influxdb UI runs by default on port 8083 (while the api runs on 8086, for which the `monitoring-influxdb` service is created). To expose this port, uncomment it in [influxdb-service.yaml](../deploy/kube-config/influxdb/influxdb-service.yaml). Alternatively, the [`kubectl port-forward`](http://kubernetes.io/docs/user-guide/kubectl/kubectl_port-forward/) command can be used, e.g. as `port-forward --namespace=kube-system monitoring-influxdb-3276295126-joqo2 8083:8083 8086:8086`. For anything more than debugging purposes, the service should be used.

3. If you find InfluxDB to be using up a lot of CPU or memory, consider placing resource restrictions on the `InfluxDB & Grafana` pod. You can add `cpu: <millicores>` and `memory: <bytes>` in the [Controller Spec](../deploy/kube-config/influxdb/influxdb-grafana-controller.yaml) and relaunch the controllers by running `kubectl apply -f deploy/kube-config/influxdb/influxdb-grafana-controller.yaml` and deleting and old influxdb pods.
