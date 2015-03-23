#!/bin/bash

set -ex
EXTRA_ARGS=""
if [ ! -z "$FLAGS" ]; then
  EXTRA_ARGS="$FLAGS"
fi

# If in Kubernetes, target the master.
if [ ! -z $KUBERNETES_RO_SERVICE_HOST ]; then
  EXTRA_ARGS="--kubernetes_master ${KUBERNETES_RO_SERVICE_HOST}:${KUBERNETES_RO_SERVICE_PORT} $EXTRA_ARGS"
fi

HEAPSTER="/usr/bin/heapster"

case $SINK in
  'influxdb') 
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
      $HEAPSTER --sink_influxdb_host $INFLUXDB_ADDRESS $EXTRA_ARGS
    elif [ ! -z $INFLUXDB_HOST ]; then
      $HEAPSTER --sink_influxdb_host ${INFLUXDB_HOST} $EXTRA_ARGS
    else
      echo "Influxdb host invalid."
      exit 1
    fi
    ;;
  'gcm') $HEAPSTER --sink gcm $EXTRA_ARGS
    ;;
  *) $HEAPSTER $EXTRA_ARGS
    ;;
esac
