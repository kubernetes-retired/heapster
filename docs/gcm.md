## Running Heapster in Kubernetes with a Google Cloud Monitoring backend

The Google Cloud Monitoring backend only works in Google Compute Engine today.

### Setup a Kubernetes cluster
[Bring up a Kubernetes cluster](https://github.com/GoogleCloudPlatform/kubernetes), if you haven't already. Ensure that `kubecfg.sh` is exported.

### Start all of the pods and services

Start the Heapster service on the cluster:

```shell
$ kubectl.sh create -f deploy/kube-config/gcm/
```

Heapster metrics are now being exported to Google Cloud Monitoring as custom metrics.

### Metrics Dashboard
To access the Google Cloud Monitoring dashboard go to: [https://app.google.stackdriver.com/](https://app.google.stackdriver.com/). Create a new dashboard and add the desired charts. Select the *Custom Metric* Resource Type and all Heapster metrics are under the `kubernetes.io` namespace. You can narrow down the query by the metric labels provided.

It is also possible to query the Google Cloud Monitoring data directly using their [custom metric read API](https://cloud.google.com/monitoring/v2beta2/timeseries/list).
