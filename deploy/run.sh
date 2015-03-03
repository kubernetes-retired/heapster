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

# If in Kubernetes, target the master.
if [ ! -z $KUBERNETES_RO_SERVICE_HOST ]; then
  EXTRA_ARGS="$EXTRA_ARGS --kubernetes_master ${KUBERNETES_RO_SERVICE_HOST}:${KUBERNETES_RO_SERVICE_PORT}"
fi

# Select what source sink to use. Options: "influxdb,gcm"
# Default is InfluxDB.
if [ -x $SINK ]; then
  SINK="influxdb"
fi

HEAPSTER="/usr/bin/heapster $EXTRA_ARGS "

if [ "$SINK" == "influxdb" ]; then
  HEAPSTER="$HEAPSTER --sink influxdb"

  # Check if in Kubernetes.
  if [ ! -z $KUBERNETES_RO_SERVICE_HOST ]; then
    # TODO(vishh): add support for passing in user name and password.
    INFLUXDB_ADDRESS=""
    if [ ! -z $MONITORING_INFLUXDB_SERVICE_HOST ]; then
      INFLUXDB_ADDRESS="${MONITORING_INFLUXDB_SERVICE_HOST}:${MONITORING_INFLUXDB_SERVICE_PORT}"
    elif [ ! -z $INFLUXDB_HOST ]; then
      INFLUXDB_ADDRESS=${INFLUXDB_HOST}
    else
      echo "InfluxDB service address not specified. Exiting."
      exit 1
    fi
    $HEAPSTER --sink_influxdb_host $INFLUXDB_ADDRESS
  elif [ ! -z $INFLUXDB_HOST ]; then
    $HEAPSTER --sink_influxdb_host ${INFLUXDB_HOST}
  else
    $HEAPSTER
  fi
elif [ "$SINK" == "gcm" ]; then
  $HEAPSTER --sink gcm
else
  $HEAPSTER
fi
