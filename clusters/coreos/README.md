Runnning Heapster on CoreOS
================================

Heapster enables cluster monitoring in a CoreOS cluster using [cAdvisor](https://github.com/google/cadvisor). 

NOTE: Some of the following steps should be handled using fleet and unit files.

**Step 1: Start cAdvisor on all hosts by default**

gce-cloud-config-with-cadvisor.yml contains 'cadvisor.service' which can be specified as part of your cloud config to bring up cAdvisor by default on all hosts.

**Step 2: Start InfluxDB and Grafana**

On a CoreOS machine start InfluxDB and grafana

```shell
$ ../../influx-grafana/influxdb/start
```

Follow the instructions in ../../influx-grafana/grafana/README.md to start a grafana container that will serve as a dashboard for the InfluxDB instance just started.

**Step 3: Start heapster coreos buddy**

Heapster does not natively support CoreOS. To help with discovering all the hosts in the cluster start a heapster buddy container that will handle discovery.

```shell
$ mkdir heapster
$ docker run --name heapster-buddy --net host -d -v $PWD/heapster:/var/run/heapster vish/heapster-buddy-coreos
```

**Step 4: Start heapster**

Figure out the host where InfluxDB is running and pass that to heapster through 'INFLUXDB_HOST' environment variable.

```shell
$ docker run --name heapster -d -e INFLUXDB_HOST=<ip>:8086 -v $PWD/heapster:/var/run/heapster vish/heapster
```

It will help to make InfluxDB and grafana a native CoreOS service, and modify heapster to auto discover the InfluxDB service as it is done in the case of Kubernetes. 
