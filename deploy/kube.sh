#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )/kube-config/influxdb"

if [[ $(which kubectl.sh) ]]; then
  KUBECTL='kubectl.sh'
elif [[ $(which kubectl) ]]; then
  KUBECTL="kubectl"
else
  echo "This script requires kubectl or kubectl.sh"
  exit 1
fi

start() {
  if ${KUBECTL} create -f "$DIR/" &> /dev/null; then
    echo "heapster pods have been setup"
  else 
    echo "failed to setup heapster pods"
  fi
}

stop() {
  ${KUBECTL} stop replicationController monitoring-influx-grafana-controller &> /dev/null
  ${KUBECTL} stop replicationController monitoring-heapster-controller &> /dev/null
  # wait for the pods to disappear.
  while ${KUBECTL} get pods -l "name=influxGrafana" -o template --template {{range.items}}{{.id}}:{{end}} | grep -c . &> /dev/null \
    || ${KUBECTL} get pods -l "name=heapster" -o template --template {{range.items}}{{.id}}:{{end}} | grep -c . &> /dev/null; do
    sleep 2
  done
  ${KUBECTL} delete -f "$DIR/" &> /dev/null || true
  echo "heapster pods have been removed."
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    stop
    start
    ;;
  *)
    echo "Usage: $0 {start|stop|restart}"
    ;;
esac

exit 0
