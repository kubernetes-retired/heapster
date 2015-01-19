#!/bin/bash

set -o errexit

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

start() {
  if kubectl.sh create -f "$DIR/" &> /dev/null; then
    echo "heapster pods have been setup"
  else 
    echo "failed to setup heapster pods"
  fi
}

stop() {
    if kubecfg.sh resize monitoring-influxGrafanaController 0 &> /dev/null \
      && kubecfg.sh resize monitoring-heapsterController 0 &> /dev/null \
      && kubectl.sh delete -f "$DIR/" &> /dev/null; then 
      echo "heapster pods have been removed."
    else
      echo "failed to remove heapster pods"
    fi
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
