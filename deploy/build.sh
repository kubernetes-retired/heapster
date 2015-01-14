#!/bin/bash

set -e

godep go build -a github.com/GoogleCloudPlatform/heapster

docker build -t kubernetes/heapster:canary $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
