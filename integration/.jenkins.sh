#!/bin/bash
set -x
SUPPORTED_KUBE_VERSIONS="0.9.0"

export GOPATH="$JENKINS_HOME/workspace/project"
export GOBIN="$GOPATH/bin"

deploy/build-test.sh \
&& influx-grafana/grafana/build-test.sh \
&& influx-grafana/influxdb/build-test.sh \
&& pushd integration \
&& godep go test -a -v --vmodule=*=1 --timeout=30m --kube_versions=$SUPPORTED_KUBE_VERSIONS github.com/GoogleCloudPlatform/heapster/integration/... \
&& popd
