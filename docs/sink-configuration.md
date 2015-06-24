Configuring sources
===================

Heapster can store data into different backends (sinks). These are specified on the command line
via the `--sink` flag. The flag takes an argument of the form `PREFIX:CONFIG[?OPTIONS]`.
Options (optional!) are specified as URL query parameters, separated by `&` as normal.
This allows each source to have custom configuration passed to it without needing to
continually add new flags to Heapster as new sinks are added. This also means
heapster can store data intomultiple sinks at once.

## Current sinks
### InfluxDB
This sink supports both monitoring metrics and events.
To use the InfluxDB sink add the following flag:
```
--sink=influxdb:<INFLUXDB_URL>[?<INFLUXDB_OPTIONS>]
```

If you're running Heapster in a Kubernetes cluster with the default InfluxDB + Grafana setup you can use the following flag:

```
--sink=influxdb:http:://monitoring-influxdb:80/
```

The following options are available:
* `user` - InfluxDB username (default: `root`)
* `pw` - InfluxDB password (default: `root`)
* `db` - InfluxDB Database name (default: `k8s`)
* `avoidColumns` - When set to `true`, labels are folded into the series name (default: `false`)


### Google Cloud Monitoring
This sink supports monitoring metrics only.
To use the InfluxDB sink add the following flag:
```
--sink=gcm[?<GCM_OPTIONS>]
```

*Note: This sink works only on a Google Compute Enginer VM as of now*

The following options are available:
* `nodeOnly` - export metrics for nodes only (default: `false`)


### Google Cloud Logging
This sink supports events only.
To use the InfluxDB sink add the following flag:
```
--sink=gcl
```

*Note: This sink works only on a Google Compute Enginer VM as of now*

This sink does not export any options!

### Hawkular-Metrics
This sink supports monitoring metrics only.
To use the Hawkular-Metrics sink add the following flag:

```
--sink=hawkular:<HAWKULAR_SERVER_URL>[?<OPTIONS>]
```

If `HAWKULAR_SERVER_URL` includes any path, the default `hawkular/metrics` is overridden.

The following options are available:

* `tenant` - Hawkular-Metrics tenantId (default: `heapster`)
