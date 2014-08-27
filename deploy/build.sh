#!/bin/bash

set -e
set -x

# Statically build cAdvisor from source and stage it.
go build -a --ldflags '-extldflags "-static"' github.com/GoogleCloudPlatform/heapster
