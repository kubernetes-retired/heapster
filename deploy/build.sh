#!/bin/bash

set -e
set -x

# Statically build cAdvisor from source and stage it.
go build -a github.com/GoogleCloudPlatform/heapster
