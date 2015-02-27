#!/bin/bash
set -x

if ! git diff --name-only origin/master | grep -c -E "*.go|*.sh|.*yaml" &> /dev/null; then
  echo "This PR does not touch files that require integration testing. Skipping integration tests!"
  exit 0
fi

SUPPORTED_KUBE_VERSIONS="0.11.0"
TEST_NAMESPACE="default"
export GOPATH="$JENKINS_HOME/workspace/project"
export GOBIN="$GOPATH/bin"

deploy/build-test.sh \
&& influx-grafana/grafana/build-test.sh \
&& influx-grafana/influxdb/build-test.sh \
&& cd integration \
&& godep go test -a -v --vmodule=*=1 --timeout=30m --namespace=$TEST_NAMESPACE --kube_versions=$SUPPORTED_KUBE_VERSIONS github.com/GoogleCloudPlatform/heapster/integration/...;
result=$?;
docker rmi -f `docker images -a -q`;
exit $result 
