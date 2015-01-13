#! /bin/bash

docker build -t kubernetes/heapster_influxdb:canary $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
