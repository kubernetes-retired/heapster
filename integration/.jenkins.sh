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

go get github.com/progrium/go-extpoints
go generate
cd integration
godep go test -a -v --vmodule=*=1 --timeout=30m --namespace=$TEST_NAMESPACE --kube_versions=$SUPPORTED_KUBE_VERSIONS github.com/GoogleCloudPlatform/heapster/...
