#!/bin/bash

set -e

pushd $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
godep go build -a 

docker build -t heapster-buddy-coreos .
popd
