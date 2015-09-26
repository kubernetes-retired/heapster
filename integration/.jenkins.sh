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

make test-unit test-integration
