Configuring sources
===================

Heapster can get data from multiple sources. These are specified on the command line
via the `--source` flag. The flag takes an argument of the form `PREFIX:CONFIG[?OPTIONS]`.
Options (optional!) are specified as URL query parameters, separated by `&` as normal.
This allows each source to have custom configuration passed to it without needing to
continually add new flags to Heapster as new sources are added. This also means
heapster can capture metrics from multiple sources at once, potentially even multiple
Kubernetes clusters.

## Current sources
### Kubernetes
To use the kubernetes source add the following flag:

```
--source=kubernetes:<KUBERNETES_MASTER>[?<KUBERNETES_OPTIONS>]
```

If you're running Heapster in a Kubernetes pod you can use the following flag:

```
--source=kubernetes
```
Heapster requires access to `token-system-monitoring` secret to connect with the master securely.
To run without auth file, use the following config:
```
--source=kubernetes:http://kubernetes-ro?auth=""
```

The following options are available:

* `apiVersion` - API version to use to talk to Kubernetes (default: `v1beta1`)
* `insecure` - whether to trust kubernetes certificates (default: `false`)
* `kubeletPort` - kubelet port to use (default: `10255`)
* `auth` - client auth file to use (default: /etc/kubernetes/kubeConfig/kubeConfig)

### Cadvisor
Cadvisor source comes in two types: standalone & CoreOS:

#### External
External cadvisor source "discovers" hosts from the specified file. Use it like this:

```
--source=cadvisor:external[?<OPTIONS>]
```

The following options are available:

* `standalone` - only use `localhost` (default: `false`)
* `hostsFile` - file containing list of hosts to gather cadvisor metrics from (default: `/var/run/heapster/hosts`)
* `cadvisorPort` - cadvisor port to use (default: `8080`)

Here is an example: 
```shell
./heapster --source="cadvisor:external?cadvisorPort=4194"
```


#### CoreOS
CoreOS cadvisor source discovers nodes from the specified fleet endpoints. Use it like this:

```
--source=cadvisor:coreos[?<OPTIONS>]
```

The following options are available:

* `fleetEndpoint` - fleet endpoints to use. This can be specified multiple times (no default)
* `cadvisorPort` - cadvisor port to use (default: `8080`)
