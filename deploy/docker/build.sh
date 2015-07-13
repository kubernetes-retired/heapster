#!/bin/bash

IMAGE=${1-heapster:canary}

set -e

pushd $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

pushd ../..
make generate
popd

godep go build -a github.com/GoogleCloudPlatform/heapster

docker build -t $IMAGE .
popd
