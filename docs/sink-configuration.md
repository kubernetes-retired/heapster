Configuring sources
===================

Heapster can store data into different backends (sinks). These are specified on the command line
via the `--sink` flag. The flag takes an argument of the form `PREFIX:CONFIG[?OPTIONS]`.
Options (optional!) are specified as URL query parameters, separated by `&` as normal.
This allows each source to have custom configuration passed to it without needing to
continually add new flags to Heapster as new sinks are added. This also means
heapster can store data into multiple sinks at once.

## Current sinks
### InfluxDB
This sink supports both monitoring metrics and events.
To use the InfluxDB sink add the following flag:
```
--sink=influxdb:<INFLUXDB_URL>[?<INFLUXDB_OPTIONS>]
```

If you're running Heapster in a Kubernetes cluster with the default InfluxDB + Grafana setup you can use the following flag:

```
--sink=influxdb:http://monitoring-influxdb:80/
```

The following options are available:
* `user` - InfluxDB username (default: `root`)
* `pw` - InfluxDB password (default: `root`)
* `db` - InfluxDB Database name (default: `k8s`)
* `avoidColumns` - When set to `true`, labels are folded into the series name (default: `false`)


### Google Cloud Monitoring
This sink supports monitoring metrics only.
To use the GCM sink add the following flag:
```
--sink=gcm
```

*Note: This sink works only on a Google Compute Enginer VM as of now*

This sink does not export any options!

### Google Cloud Monitoring Autoscaling
This sink supports monitoring metrics for autoscaling purposes only.
To use the GCM Autoscaling sink add the following flag:
```
--sink=gcmautoscaling
```

*Note: This sink works only on a Google Compute Enginer VM as of now*

This sink does not export any options!

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

If `HAWKULAR_SERVER_URL` includes any path, the default `hawkular/metrics` is overridden. To use SSL, the `HAWKULAR_SERVER_URL` has to start with `https`

The following options are available:

* `tenant` - Hawkular-Metrics tenantId (default: `heapster`)
* `labelToTenant` - Hawkular-Metrics uses given label's value as tenant value when storing data
* `useServiceAccount` - Sink will use the service account token to authorize to Hawkular-Metrics (requires Openshift)
* `insecure` - SSL connection will not verify the certificates
* `caCert` - A path to the CA Certificate file that will be used in the connection
* `auth` - Kubernetes authentication file that will be used for constructing the TLSConfig

A combination of `insecure` / `caCert` / `auth` is not supported, only a single of these parameters is allowed at once. 

## Modifying the sinks at runtime

Using the `/api/v1/sinks` endpoint, it is possible to fetch the sinks
currently in use via a GET request or to change them via a POST request. The
format is the same as when passed via command line flags.

For example, to set gcm and influxdb as sinks, you may do the following:

```
echo '["gcm", "influxdb:http://monitoring-influxdb:8086"]' | curl \
    --insecure -u admin:<password> -X POST -d @- \
    -H "Accept: application/json" -H "Content-Type: application/json" \
    https://<master-ip>/api/v1/proxy/namespaces/kube-system/services/monitoring-heapster/api/v1/sinks
```
