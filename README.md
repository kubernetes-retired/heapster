heapster
===========

A [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) service that does the following:

1. Discovers all minions in a Kubernetes cluster
2. Collects container statistics from the cadvisors running on the minions
2. Organizes stats into Pods
3. Stores Pod stats in a configurable backend

Along with each continer stat entry, its Pod ID, Container name, Pod IP, Hostname and Labels are also stored. Labels are stored as key:value pairs.

Supports in-memory backend and InfluxDB backends.

####In memory:
```
$ ./heapster --kubernetes_master x.x.x.x -kubernetes_master_auth admin:your_passwd
```
**Optional configuration flags**
- ```-poll_duration 5s```

####With InfluxDB:
```
$ ./heapster --kubernetes_master x.x.x.x -kubernetes_master_auth admin:your_passwd -sink influxdb -sink_influxdb_host x.x.y.z:8086
```
**Optional Flags**
- ```-sink_influxdb_username user```
- ```-sink_influxdb_password passwd```
- ```-sink_influxdb_name mydb```
- ```-sink_influxdb_buffer_duration 1m```

To setup an Influxdb Pod in Kubernetes look [here](https://github.com/vishh/grafana-influxdb-k8s)
