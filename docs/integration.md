## Heapster integration testing

Heapster can be tested against an existing Kubernetes cluster running on GCE. Tests under [integration](../integration) can be used for integration testing.

Figure out the server version of your current Kubernetes cluster.
```shell
$ kubectl.sh version
```
Use the full version specified there for integration testing. For example the full version can be `0.11.0` or `0.9.1`.

Run the integration test as follows:
```shell
$ cd integration
$ MY_VERSION=0.11.0
$ godep go test -a -v --vmodule=*=1 --timeout=30m --kube_versions=$MY_VERSION
```

Note that the test expects `~/.kubernetes_auth` file to exist. If it does not exist, which will be the case with recent kubernetes versions,
you can pass in the path to the auth file via the flag `--auth_config` or create a symlink from the new auth file to `~/.kubernetes_auth`.
```shell
$ godep go test -a -v --timeout=30m --kube_versions=$MY_VERSION --auth_config=~/.kube/<cluster-name>/kubernetes_auth
```

The tests currently use [gcutil](https://cloud.google.com/compute/docs/gcutil/). We will migrate to using [`gcloud compute`](https://cloud.google.com/compute/docs/gcloud-compute/) soon.