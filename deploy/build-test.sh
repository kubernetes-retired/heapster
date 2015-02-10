#!/bin/bash

set -e

pushd $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

godep go build -a github.com/GoogleCloudPlatform/heapster

docker build -t vish/heapster:e2e_test1 .
docker push vish/heapster:e2e_test1
popd
