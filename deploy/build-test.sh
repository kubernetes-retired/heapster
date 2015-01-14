#!/bin/bash

set -e

godep go build -a github.com/GoogleCloudPlatform/heapster

docker build -t vish/heapster:e2e_test $( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

docker push vish/heapster:e2e_test
