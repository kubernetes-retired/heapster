#!/bin/bash

set -e

/usr/bin/influxdb -config=/opt/influxdb/shared/config.toml &
if [ ! -z "${DEFAULT_DB}" ]; then    
    until false
    do
	result=$(curl -X POST 'http://localhost:8086/db?u=root&p=root' -d '{"name": ${DEFAULT_DB}}')
	echo result
	if [[ result == *exists* ]]; then
	    break
	fi	    
	sleep 5
	echo "Attempting to create db $(DEFAULT_DB) again"
    done
fi
wait
