#! /bin/bash

docker build -t kubernetes/heapster_grafana:canary $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
