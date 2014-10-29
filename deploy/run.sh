#!/bin/bash

set -e

KUBE_ARGS=""

if [ "$KUBE_MASTER" != "" -a "$KUBE_MASTER_AUTH" != "" ]; then
    echo "Detected Kube specific args. Starting in Kube mode."
    KUBE_ARGS="--kubernetes_master ${KUBE_MASTER} --kubernetes_master_auth ${KUBE_MASTER_AUTH}"
fi

# Check if InfluxDB service is running
if [ ! -z $INFLUX_MASTER_SERVICE_PORT ]; then
# TODO(vishh): add support for passing in user name and password.    
    /usr/bin/heapster $KUBE_ARGS --sink influxdb --sink_influxdb_host "${INFLUX_MASTER_SERVICE_HOST}:${INFLUX_MASTER_SERVICE_PORT}"
elif [ ! -z $INFLUXDB_HOST ]; then
    /usr/bin/heapster $KUBE_ARGS --sink influxdb --sink_influxdb_host ${INFLUXDB_HOST}
else
    /usr/bin/heapster $KUBE_ARGS
fi
