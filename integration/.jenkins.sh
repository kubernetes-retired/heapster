#!/bin/bash
set -x

export GOPATH="$JENKINS_HOME/workspace/project"
export GOBIN="$GOPATH/bin"

if ! git diff --name-only origin/master | grep -c -E "*.go|*.sh|.*yaml" &> /dev/null; then
  echo "This PR does not touch files that require integration testing. Skipping integration tests!"
  exit 0
fi

SUPPORTED_KUBE_VERSIONS="0.14.1"
TEST_NAMESPACE="default"

cd integration
godep go test -a -v --vmodule=*=1 --timeout=30m --namespace=$TEST_NAMESPACE --kube_versions=$SUPPORTED_KUBE_VERSIONS github.com/GoogleCloudPlatform/heapster/...
