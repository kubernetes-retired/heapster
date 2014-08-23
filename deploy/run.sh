#!/bin/bash

set -e

if [ ! ${KUBE_MASTER:?} ]; then
    echo "KUBE_MASTER env variable is not set"
    exit 1
fi
if [ ! ${KUBE_MASTER_AUTH:?} ]; then
    echo "KUBE_MASTER_AUTH env variable is not set"
    exit 1
fi

# Check if InfluxDB service is running
if [ ${INFLUX_MASTER_PORT_0_TCP_ADDR:?} ]; then
# add support for passing in user name and password.    
    echo "InfluxDB service discovered. Starting heapster with InfluxDb sink running at ${INFLUX_MASTER_PORT_0_TCP_ADDR}:${INFLUX_MASTER_SERVICE_PORT}"
    /usr/bin/heapster --kubernetes_master ${KUBE_MASTER} --kubernetes_master_auth ${KUBE_MASTER_AUTH} --sink influxdb --sink_influxdb_host "${INFLUX_MASTER_PORT_0_TCP_ADDR}:${INFLUX_MASTER_SERVICE_PORT}"
elif [ ${INFLUXDB_HOST:?} ]; then
    echo "InfluxDB host specified in commandline. Starting heapster with InfluxDb sink running at ${INFLUXDB_HOST}"
    /usr/bin/heapster --kubernetes_master ${KUBE_MASTER} --kubernetes_master_auth ${KUBE_MASTER_AUTH} --sink influxdb --sink_influxdb_host ${INFLUXDB_HOST}
else
    echo "Starting heapster with in-memory sink"
    /usr/bin/heapster --kubernetes_master ${KUBE_MASTER} --kubernetes_master_auth ${KUBE_MASTER_AUTH}
fi