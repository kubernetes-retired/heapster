#! /bin/bash

pushd $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
docker build -t kubernetes/heapster_grafana:canary .
popd
