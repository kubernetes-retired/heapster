#!/bin/bash

set -e

pushd $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
godep go build -a github.com/GoogleCloudPlatform/heapster

docker build -t heapster:canary .
popd
