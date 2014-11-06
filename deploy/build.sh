#!/bin/bash

set -e

godep go build -a github.com/GoogleCloudPlatform/heapster

docker build -t kubernetes/heapster:canary .

docker push kubernetes/heapster:canary
