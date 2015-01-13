#! /bin/bash

docker build -t vish/heapster_influxdb:e2e_test $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
docker push vish/heapster_influxdb:e2e_test

