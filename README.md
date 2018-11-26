# Heapster

***RETIRED***: Heapster is now retired.  See the [deprecation timeline](docs/deprecation.md)
for more information on support. We will not be making changes to Heapster.

The following are potential migration paths for Heapster functionality:

- **For basic CPU/memory HPA metrics**: Use [metrics-server](https://github.com/kubernetes-incubator/metrics-server).

- **For general monitoring**: Consider a third-party monitoring pipeline that can gather Prometheus-formatted metrics.
  The kubelet exposes all the metrics exported by Heapster in Prometheus format.
  One such monitoring pipeline can be set up using the [Prometheus Operator](https://github.com/coreos/prometheus-operator), which
  deploys Prometheus itself for this purpose.

- **For event transfer**: Several third-party tools exist to transfer/archive Kubernetes events, depending on your sink.
  [heptiolabs/eventrouter](https://github.com/heptiolabs/eventrouter) has been suggested as a general alternative.

[![GoDoc](https://godoc.org/k8s.io/heapster?status.svg)](https://godoc.org/k8s.io/heapster) [![Build Status](https://travis-ci.org/kubernetes/heapster.svg?branch=master)](https://travis-ci.org/kubernetes/heapster)  [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/heapster)](https://goreportcard.com/report/github.com/kubernetes/heapster)

Heapster enables Container Cluster Monitoring and Performance Analysis for [Kubernetes](https://github.com/kubernetes/kubernetes) (versions v1.0.6 and higher), and platforms which include it.

Heapster collects and interprets various signals like compute resource usage, lifecycle events, etc.
Note that the model API, formerly used provide REST access to its collected metrics, is now deprecated.
Please see [the model documentation](docs/model.md) for more details.

Heapster supports multiple sources of data.
More information [here](docs/source-configuration.md).

Heapster supports the pluggable storage backends described [here](docs/sink-owners.md).
We welcome patches that add additional storage backends.
Documentation on storage sinks [here](docs/sink-configuration.md).
The current version of Storage Schema is documented [here](docs/storage-schema.md).

### Running Heapster on Kubernetes

Heapster can run on a Kubernetes cluster using a number of backends.  Some common choices:
- [InfluxDB](docs/influxdb.md)
- [Stackdriver Monitoring and Logging](docs/google.md) for Google Cloud Platform
- [Other backends](docs/)

### Running Heapster on OpenShift

Using Heapster to monitor an OpenShift cluster requires some additional changes to the Kubernetes instructions to allow communication between the Heapster instance and OpenShift's secured endpoints. To run standalone Heapster or a combination of Heapster and Hawkular-Metrics in OpenShift, follow [this guide](https://github.com/openshift/origin-metrics).

#### Troubleshooting guide [here](docs/debugging.md)


## Community

Contributions, questions, and comments are all welcomed and encouraged! Developers hang out on [Slack](https://kubernetes.slack.com) in the #sig-instrumentation channel (get an invitation [here](http://slack.kubernetes.io/)). We also have the [kubernetes-dev Google Groups mailing list](https://groups.google.com/forum/#!forum/kubernetes-dev). If you are posting to the list please prefix your subject with "heapster: ".
