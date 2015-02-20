#!/bin/bash

set -ex
EXTRA_ARGS=""
if [ ! -z $COREOS ]; then
  EXTRA_ARGS="$EXTRA_ARGS --coreos"
fi
if [ ! -z $DEBUG ]; then
  EXTRA_ARGS="$EXTRA_ARGS --vmodule=*=3"
fi
if [ ! -x $CADVISOR_PORT ]; then
  EXTRA_ARGS="$EXTRA_ARGS --cadvisor_port=$CADVISOR_PORT"
fi

HEAPSTER="/usr/bin/heapster $EXTRA_ARGS "
# Check if InfluxDB service is running
if [ ! -z $KUBERNETES_RO_SERVICE_HOST ]; then
  # TODO(vishh): add support for passing in user name and password.    
  INFLUXDB_ADDRESS=""
  if [ ! -z $MONITORING_INFLUXDB_SERVICE_HOST ]; then
    INFLUXDB_ADDRESS="${MONITORING_INFLUXDB_SERVICE_HOST}:${MONITORING_INFLUXDB_SERVICE_PORT}"
  elif [ ! -z $INFLUXDB_HOST ]; then
    INFLUXDB_ADDRESS=${INFLUXDB_HOST}
  else 
    echo "InfluxDB service address not found. Exiting."
    exit 1
  fi
  $HEAPSTER --kubernetes_master "${KUBERNETES_RO_SERVICE_HOST}:${KUBERNETES_RO_SERVICE_PORT}" --sink influxdb --sink_influxdb_host $INFLUXDB_ADDRESS
elif [ ! -z $INFLUXDB_HOST ]; then
  $HEAPSTER --sink influxdb --sink_influxdb_host ${INFLUXDB_HOST}
else
  $HEAPSTER
fi
