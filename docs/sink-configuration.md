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
This sink supports both monitoring metrics only.
To use the InfluxDB sink add the following flag:
```
--sink=influxdb:gcm
```

*Note: This sink works only on a Google Compute Enginer VM as of now*

This sink does not export any options!


### Google Cloud Logging
This sink supports events only.
To use the InfluxDB sink add the following flag:
```
--sink=influxdb:gcl
```

*Note: This sink works only on a Google Compute Enginer VM as of now*

This sink does not export any options!

### Bosun
This sink supports monitoring metrics only as of now.
To use the Bosun sink add the following flag:
```
--sink=bosun:<BOSUN_URL>[?<BOSUN_OPTIONS>]
```

Example:
```
--sink=bosun:http://localhost:8070/
```

This sink does not export any options as of now!




