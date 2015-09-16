#!/bin/bash

set -e -x

export GOPATH="$JENKINS_HOME/workspace/project"
export GOBIN="$GOPATH/bin"
export PATH="$GOBIN:$PATH"

rm -rf $GOPATH/src/k8s.io
mkdir -p $GOPATH/src/k8s.io
mv $GOPATH/src/github.com/GoogleCloudPlatform/heapster $GOPATH/src/k8s.io/heapster

if ! git diff --name-only origin/master | grep -c -E "*.go|*.sh|.*yaml" &> /dev/null; then
  echo "This PR does not touch files that require integration testing. Skipping integration tests!"
  exit 0
fi

SUPPORTED_KUBE_VERSIONS="1.0.5"
TEST_NAMESPACE="heapster-e2e-tests"

make test-unit
godep go test -a -v --timeout=30m k8s.io/heapster/integration/... --vmodule=*=2 --namespace=$TEST_NAMESPACE --kube_versions=$SUPPORTED_KUBE_VERSIONS 
