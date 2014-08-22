cAggregator
===========

Simple [Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes) service that does the following:
1. Periodically collects container statistics from all the cadvisors in a kubernetes cluster
2. Organizes the stats data into Pods
3. Stores them in a configurable DB. 

Along with each continer stat entry, its pod ID, logical name, pod IP, Hostname and labels are also stored.

Supports in-memory backend and InfluxDB backends.

```$ ./caggregator --kubernetes_master x.x.x.x -kubernetes_master_auth admin:your_passwd```

**Optional configuration flags**
```-poll_duration 5s```

To run with **InfluxDB** backend,

**Required Flags**
```-sink influxdb -sink_influxdb_host x.x.y.z:8086```

**Optional Flags**
```-sink_influxdb_username user```
```-sink_influxdb_password passwd```
```-sink_influxdb_name mydb```
```-sink_influxdb_buffer_duration 1m```

To setup an Influxdb Pod in Kubernetes look [here](https://github.com/vishh/grafana-influxdb-k8s)

TODO:
1. Create a Docker image
2. Create a Kubernetes Container Manifest
3. Auto discover influxdb service
4. Cache data in-memory when influxdb service is down.
