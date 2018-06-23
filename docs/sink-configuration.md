Configuring sinks
=================

Heapster can store data into different backends (sinks). These are specified on the command line
via the `--sink` flag. The flag takes an argument of the form `PREFIX:CONFIG[?OPTIONS]`.
Options (optional!) are specified as URL query parameters, separated by `&` as normal.
This allows each source to have custom configuration passed to it without needing to
continually add new flags to Heapster as new sinks are added. Heapster can 
store data into multiple sinks at once if multiple `--sink` flags are specified.

## Current sinks

### Log

This sink writes all data to the standard output which is particularly useful for debugging.

    --sink=log

### InfluxDB
This sink supports both monitoring metrics and events.
*This sink supports InfluxDB versions v0.9 and above*.
To use the InfluxDB sink add the following flag:

	--sink=influxdb:<INFLUXDB_URL>[?<INFLUXDB_OPTIONS>]

If you're running Heapster in a Kubernetes cluster with the default InfluxDB + Grafana setup you can use the following flag:

	--sink=influxdb:http://monitoring-influxdb:80/

The following options are available:
* `user` - InfluxDB username (default: `root`)
* `pw` - InfluxDB password (default: `root`)
* `db` - InfluxDB Database name (default: `k8s`)
* `retention` - Duration of the default InfluxDB retention policy, e.g. `4h` or `7d` (default: `0` meaning infinite)
* `secure` - Connect securely to InfluxDB (default: `false`)
* `insecuressl` - Ignore SSL certificate validity (default: `false`)
* `withfields` - Use [InfluxDB fields](storage-schema.md#using-fields) (default: `false`)
* `cluster_name` - Cluster name for different Kubernetes clusters. (default: `default`)
* `disable_counter_metrics` - Disable sink counter metrics to InfluxDB. (default: `false`)
* `concurrency` - concurrency for sinking to InfluxDB. (default: `1`)

### Stackdriver

This sink supports monitoring metrics only.
To use the Stackdriver sink add following flag:

	--sink=stackdriver[:?<STACKDRIVER_OPTIONS>]

The following options are available:
* `workers` - The number of workers. (default: `1`)
* `cluster_name` - Cluster name for different Kubernetes clusters. (default: ``)

### Google Cloud Monitoring
This sink supports monitoring metrics only.
To use the GCM sink add the following flag:

	--sink=gcm

*Note: This sink works only on a Google Compute Engine VM as of now*

GCM has one option - `metrics` that can be set to:
* all - the sink exports all metrics
* autoscaling - the sink exports only autoscaling-related metrics

### Google Cloud Logging
This sink supports events only.
To use the GCL sink add the following flag:

	--sink=gcl

*Notes:*
 * This sink works only on a Google Compute Engine VM as of now
 * GCE instance must have “https://www.googleapis.com/auth/logging.write” auth scope
 * GCE instance must have Logging API enabled for the project in Google Developer Console
 * GCL Logs are accessible via:
    * `https://console.developers.google.com/project/<project_ID>/logs?service=custom.googleapis.com`
    * Where `project_ID` is the project ID of the Google Cloud Platform project.
    * Select `kubernetes.io/events` from the `All logs` drop down menu.

### StatsD
This sink supports monitoring metrics only.
To use the StatsD sink add the following flag:
```
  --sink=statsd:udp://<HOST>:<PORT>[?<OPTIONS>]
```

The following options are available:

* `prefix`           - Adds specified prefix to all metrics, default is empty
* `protocolType`     - Protocol type specifies the message format, it can be either etsystatsd or influxstatsd, default is etsystatsd
* `numMetricsPerMsg` - number of metrics to be packed in an UDP message, default is 5
* `renameLabels`     - renames labels, old and new label separated by ':' and pairs of old and new labels separated by ','
* `allowedLabels`    - comma-separated labels that are allowed, default is empty ie all labels are allowed
* `labelStyle`       - convert labels from default snake case to other styles, default is no conversion. Styles supported are `lowerCamelCase` and `upperCamelCase`

For example.
```
  --sink=statsd:udp://127.0.0.1:4125?prefix=kubernetes.example.&protocolType=influxstatsd&numMetricsPerMsg=10&renameLabels=type:k8s_type,host_name:k8s_host_name&allowedLabels=container_name,namespace_name,type,host_name&labelStyle=lowerCamelCase
```

#### etsystatsd metrics format

| Metric Set Type |  Metrics Format                                                                       |
|:----------------|:--------------------------------------------------------------------------------------|
| Cluster         | `<PREFIX>.<SUFFIX>`                                                                   |
| Node            | `<PREFIX>.node.<NODE>.<SUFFIX>`                                                       |
| Namespace       | `<PREFIX>.namespace.<NAMESPACE>.<SUFFIX>`                                             |
| Pod             | `<PREFIX>.node.<NODE>.namespace.<NAMESPACE>.pod.<POD>.<SUFFIX>`                       |
| PodContainer    | `<PREFIX>.node.<NODE>.namespace.<NAMESPACE>.pod.<POD>.container.<CONTAINER>.<SUFFIX>` |
| SystemContainer | `<PREFIX>.node.<NODE>.sys-container.<SYS-CONTAINER>.<SUFFIX>`                         |

* `PREFIX`      - configured prefix
* `SUFFIX`      - `[.<USER_LABELS>].<METRIC>[.<RESOURCE_ID>]`
* `USER_LABELS` - user provided labels `[.<KEY1>.<VAL1>][.<KEY2>.<VAL2>] ...`
* `METRIC`      - metric name, eg: filesystem/usage
* `RESOURCE_ID` - An unique identifier used to differentiate multiple metrics of the same type. eg: FS partitions under filesystem/usage

#### influxstatsd metrics format
Influx StatsD is very similar to Etsy StatsD. Tags are supported by adding a comma-separated list of tags in key=value format.

```
<METRIC>[,<KEY1=VAL1>,<KEY2=VAL2>...]:<METRIC_VALUE>|<METRIC_TYPE>
```

### Hawkular-Metrics
This sink supports monitoring metrics only.
To use the Hawkular-Metrics sink add the following flag:

	--sink=hawkular:<HAWKULAR_SERVER_URL>[?<OPTIONS>]

If `HAWKULAR_SERVER_URL` includes any path, the default `hawkular/metrics` is overridden. To use SSL, the `HAWKULAR_SERVER_URL` has to start with `https`

The following options are available:

* `tenant` - Hawkular-Metrics tenantId (default: `heapster`)
* `labelToTenant` - Hawkular-Metrics uses given label's value as tenant value when storing data
* `useServiceAccount` - Sink will use the service account token to authorize to Hawkular-Metrics (requires OpenShift)
* `insecure` - SSL connection will not verify the certificates
* `caCert` - A path to the CA Certificate file that will be used in the connection
* `auth` - Kubernetes authentication file that will be used for constructing the TLSConfig
* `user` - Username to connect to the Hawkular-Metrics server
* `pass` - Password to connect to the Hawkular-Metrics server
* `filter` - Allows bypassing the store of matching metrics, any number of `filter` parameters can be given with a syntax of `filter=operation(param)`. Supported operations and their params:
  * `label` - The syntax is `label(labelName:regexp)` where `labelName` is 1:1 match and `regexp` to use for matching is given after `:` delimiter
  * `name` - The syntax is `name(regexp)` where MetricName is matched (such as `cpu/usage`) with a `regexp` filter
* `batchSize`- How many metrics are sent in each request to Hawkular-Metrics (default is 1000)
* `concurrencyLimit`- How many concurrent requests are used to send data to the Hawkular-Metrics (default is 5)
* `labelTagPrefix` - A prefix to be placed in front of each label when stored as a tag for the metric (default is `labels.`)
* `disablePreCache` - Disable cache initialization by fetching metric definitions from Hawkular-Metrics

A combination of `insecure` / `caCert` / `auth` is not supported, only a single of these parameters is allowed at once. Also, combination of `useServiceAccount` and `user` + `pass` is not supported. To increase the performance of Hawkular sink in case of multiple instances of Hawkular-Metrics (such as scaled scenario in OpenShift) modify the parameters of batchSize and concurrencyLimit to balance the load on Hawkular-Metrics instances.


### Wavefront
The Wavefront sink supports monitoring metrics only.
To use the Wavefront sink add the following flag:

    --sink=wavefront:<WAVEFRONT_PROXY_URL:PORT>[?<OPTIONS>]

The following options are available:

* `clusterName` - The name of the Kubernetes cluster being monitored. This will be added as a tag called `cluster` to metrics in Wavefront (default: `k8s-cluster`)
* `prefix` - The prefix to be added to all metrics that Heapster collects (default: `heapster.`)
* `includeLabels` - If set to true, any K8s labels will be applied to metrics as tags (default: `false`)
* `includeContainers` - If set to true, all container metrics will be sent to Wavefront. When set to false, container level metrics are skipped (pod level and above are still sent to Wavefront) (default: `true`)


### OpenTSDB
This sink supports both monitoring metrics and events.
To use the OpenTSDB sink add the following flag:

    --sink=opentsdb:<OPENTSDB_SERVER_URL>[?<OPTIONS>]

Currently, accessing OpenTSDB via its rest apis doesn't need any authentication, so you
can enable OpenTSDB sink like this:

    --sink=opentsdb:http://192.168.1.8:4242?cluster=k8s-cluster

The following options are available:

* `cluster` - The name of the Kubernetes cluster being monitored. This will be added as a tag called `cluster` to metrics in OpenTSDB (default: `k8s-cluster`)

### Kafka
This sink supports monitoring metrics only.
To use the kafka sink add the following flag:

    --sink="kafka:<?<OPTIONS>>"

Normally, kafka server has multi brokers, so brokers' list need be configured for producer.
So, we provide kafka brokers' list and topics about timeseries & topic in url's query string.
Options can be set in query string, like this:

* `brokers` - Kafka's brokers' list.
* `timeseriestopic` - Kafka's topic for timeseries. Default value : `heapster-metrics`.
* `eventstopic` - Kafka's topic for events. Default value : `heapster-events`.
* `compression` - Kafka's compression for both topics. Must be `gzip` or `none` or `snappy` or `lz4`. Default value : none.
* `user` - Kafka's SASL PLAIN username. Must be set with `password` option.
* `password` - Kafka's SASL PLAIN password. Must be set with `user` option.
* `cacert` - Kafka's SSL Certificate Authority file path.
* `cert` - Kafka's SSL Client Certificate file path (In case of Two-way SSL). Must be set with `key` option.
* `key` - Kafka's SSL Client Private Key file path (In case of Two-way SSL). Must be set with `cert` option.
* `insecuressl` - Kafka's Ignore SSL certificate validity. Default value : `false`.

For example,

    --sink="kafka:?brokers=localhost:9092&brokers=localhost:9093&timeseriestopic=testseries"
    or
    --sink="kafka:?brokers=localhost:9092&brokers=localhost:9093&eventstopic=testtopic"

### Riemann
This sink supports monitoring metrics and events.
To use the Riemann sink add the following flag:

	--sink="riemann:<RIEMANN_SERVER_URL>[?<OPTIONS>]"

The following options are available:

* `ttl` - TTL for writing to Riemann. Default: `60 seconds`
* `state` - The event state. Default: `""`
* `tags` - Default. `heapster`
* `batchsize` - The Riemann sink sends batch of events. The default size is `1000`

For example,

--sink=riemann:http://localhost:5555?ttl=120&state=ok&tags=foobar&batchsize=150

### Elasticsearch
This sink supports monitoring metrics and events. To use the Elasticsearch
sink add the following flag:
```
    --sink=elasticsearch:<ES_SERVER_URL>[?<OPTIONS>]
```
Normally an Elasticsearch cluster has multiple nodes or a proxy, so these need
to be configured for the Elasticsearch sink. To do this, you can set
`ES_SERVER_URL` to a dummy value, and use the `?nodes=` query value for each
additional node in the cluster. For example:
```
  --sink=elasticsearch:?nodes=http://foo.com:9200&nodes=http://bar.com:9200
```
(*) Notice that using the `?nodes` notation will override the `ES_SERVER_URL`

If you run your ElasticSearch cluster behind a loadbalancer (or otherwise do
not want to specify multiple nodes) then you can do the following:
```
  --sink=elasticsearch:http://elasticsearch.example.com:9200?sniff=false
```

Besides this, the following options can be set in query string:

(*) Note that the keys are case sensitive

* `index` - the index for metrics and events. The default is `heapster`
* `esUserName` - the username if authentication is enabled
* `esUserSecret` - the password if authentication is enabled
* `maxRetries` - the number of retries that the Elastic client will perform
  for a single request after before giving up and return an error. It is `0`
  by default, so retry is disabled by default.
* `healthCheck` - specifies if healthchecks are enabled by default. It is enabled
  by default. To disable, provide a negative boolean value like `0` or `false`.
* `sniff` - specifies if the sniffer is enabled by default. It is enabled
  by default. To disable, provide a negative boolean value like `0` or `false`.
* `startupHealthcheckTimeout` - the time in seconds the healthcheck waits for
  a response from Elasticsearch on startup, i.e. when creating a client. The
  default value is `1`.
* `ver` - ElasticSearch cluster version, can be either `2` or `5`. The default is `5`
* `bulkWorkers` - number of workers for bulk processing. Default value is `5`.
* `cluster_name` - cluster name for different Kubernetes clusters. Default value is `default`.
* `pipeline` - (optional; >ES5) Ingest Pipeline to process the documents. The default is disabled(empty value)

#### AWS Integration
In order to use AWS Managed Elastic we need to use one of the following methods:

1. Making sure the public IPs of the Heapster are allowed on the Elasticsearch's Access Policy

-OR-

2. Configuring an Access Policy with IAM
	1. Configure the Elasticsearch cluster policy with IAM User
	2. Create a secret that stores the IAM credentials
	3. Expose the credentials to the environment variables: `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`

	```
	env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: aws-heapster
            key: aws.id
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: aws-heapster
            key: aws.secret
	```

### Graphite/Carbon
This sink supports monitoring metrics only.
To use the graphite sink add the following flag:

    --sink="graphite:<PROTOCOL>://<HOST>[:<PORT>][<?<OPTIONS>>]"

PROTOCOL must be `tcp` or `udp`, PORT is 2003 by default.

These options are available:
* `prefix` - Adds specified prefix to all metric paths

For example,

    --sink="graphite:tcp://metrics.example.com:2003?prefix=kubernetes.example"

Metrics are sent to Graphite with this hierarchy:
* `PREFIX`
  * `cluster`
  * `namespaces`
    * `NAMESPACE`
  * `nodes`
    * `NODE`
      * `pods`
        * `NAMESPACE`
           * `POD`
             * `containers`
               * `CONTAINER`
      * `sys-containers`
        * `SYS-CONTAINER`

### Librato

This sink supports monitoring metrics only.

To use the librato sink add the following flag:

    --sink=librato:<?<OPTIONS>>

Options can be set in query string, like this:

* `username` - Librato user email address (https://www.librato.com/docs/api/#authentication).
* `token` - Librato API token
* `prefix` - Prefix for all measurement names
* `tags` - By default provided tags (comma separated list)
* `tag_{name}` - Value for the tag `name`

For example,

    --sink=librato:?username=xyz&token=secret&prefix=k8s&tags=cluster&tag_cluster=staging

The librato sink currently only works with accounts, which support [tagged metrics](https://www.librato.com/docs/kb/faq/account_questions/tags_or_sources/).

### Honeycomb

This sink supports both monitoring metrics and events.

To use the Honeycomb sink add the following flag:

    --sink="honeycomb:<?<OPTIONS>>"

Options can be set in query string, like this:

* `dataset` - Honeycomb Dataset to which to publish metrics/events
* `writekey` - Honeycomb Write Key for your account
* `apihost` - Option to send metrics to a different host (default: https://api.honeycomb.com) (optional)

For example,

    --sink="honeycomb:?dataset=mydataset&writekey=secretwritekey"

## Using multiple sinks

Heapster can be configured to send k8s metrics and events to multiple sinks by specifying the`--sink=...` flag multiple times.

For example, to send data to both gcm and influxdb at the same time, you can use the following:

```shell
    --sink=gcm --sink=influxdb:http://monitoring-influxdb:80/
```
