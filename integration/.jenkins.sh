#!/bin/bash
set -x

export GOPATH="$JENKINS_HOME/workspace/project"
export GOBIN="$GOPATH/bin"
export PATH="$GOBIN:$PATH"

if ! git diff --name-only origin/master | grep -c -E "*.go|*.sh|.*yaml" &> /dev/null; then
  echo "This PR does not touch files that require integration testing. Skipping integration tests!"
  exit 0
fi

SUPPORTED_KUBE_VERSIONS="0.15.0"
TEST_NAMESPACE="default"

make test-unit
godep go test -a -v --timeout=30m github.com/GoogleCloudPlatform/heapster/integration/... --vmodule=*=1 --namespace=$TEST_NAMESPACE --kube_versions=$SUPPORTED_KUBE_VERSIONS 
