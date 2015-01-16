#!/bin/bash
set -x
export GOPATH="$JENKINS_HOME/workspace/project"
export GOBIN="$GOPATH/bin"

deploy/build-test.sh \
&& influx-grafana/grafana/build-test.sh \
&& influx-grafana/influxdb/build-test.sh \
&& pushd integration \
&& cd integration \
&& godep go test -a -v --vmodule=*=1 --timeout=30m --kube_versions="0.7.2,0.8.0" ./... \
&& popd
