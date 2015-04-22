## Metrics

Heapster exports the following metrics to its backends.

| Metric Name        | Description                                                                                        | Type       | Units       | Supported Since |
|--------------------|----------------------------------------------------------------------------------------------------|------------|-------------|-----------------|
| uptime             | Number of millisecond since the container was started                                             | Cumulative | Milliseconds | v0.9            |
| cpu/usage          | Cumulative CPU usage on all cores                                                                  | Cumulative | Nanoseconds       | v0.9            |
| memory/usage       | Total memory usage                                                                                 | Gauge      | Bytes       | v0.9            |
| memory/page_faults | Number of major page faults                                                                        | Cumulative      | Count       | v0.9            |
| memory/working_set | Total working set usage. Working set is the memory being used and not easily dropped by the Kernel | Gauge      | Bytes       | v0.9            |
| network/rx         | Cumulative number of bytes received over the network                                               | Cumulative | Bytes       | v0.9            |
| network/rx_errors  | Cumulative number of errors while receiving over the network                                       | Cumulative | Count       | v0.9            |
| network/tx         | Cumulative number of bytes sent over the network                                                   | Cumulative | Bytes       | v0.9            |
| network/tx_errors  | Cumulative number of errors while sending over the network                                         | Cumulative | Count       | v0.9            |
| filesystem/usage   | Total number of bytes used on a filesystem identified by label 'resource_id'                       | Gauge      | Bytes       | HEAD            |

*Note: Gauge refers to instantaneous metrics*

## Labels

Heapster tags each metric with the following labels.

| Label Name     | Description                                                                   | Supported Since | Kubernetes specific |
|----------------|-------------------------------------------------------------------------------|-----------------|---------------------|
| pod_id         | Unique ID of a Pod                                                            | v0.9            | Yes                 |
| pod_namespace  | The namespace of a Pod                                                        | v0.10           | Yes               |
| container_name | User-provided name of the container or full cgroup name for system containers | v0.9            | No                  |
| labels         | Comma-separated list of user-provided labels. Format is 'key:value'           | v0.9            | Yes                 |
| hostname       | Hostname where the container ran                                              | v0.9            | No                  |
| resource_id    | An unique identifier used to differentiate multiple metrics of the same type. e.x. Fs partitions under filesystem/usage | HEAD | No |


## Storage Schema

### InfluxDB

Each metric translates to a separate 'series' in InfluxDB. Labels are stored as additional columns.
The series name is constructed by combining the metric name with its type and units: "metric Name"_"units"_"type".

###### Query
`list series`

###### Output
```
cpu/usage_ns_cumulative
filesystem/usage
memory/page_faults_gauge
memory/usage_bytes_gauge
memory/working_set_bytes_gauge
network/rx_bytes_cumulative
network/rx_errors_cumulative
network/tx_bytes_cumulative
network/tx_errors_cumulative
uptime_ms_cumulative
```
*Note: Unit 'Count' is ignored*

Heapster adds timestamp and sequence number to every metric.

### Google Cloud Monitoring

Metrics mentioned above are stored along with corresponding labels as custom metrics in Google Cloud Monitoring.
Each metric is pushed with an additonal namespace prefix `kubernetes.io`.
TODO: Add more information about GCM support.
