#!/bin/bash

set -e

if [ "${ELASTICSEARCH_HOST}" == "**None**" ]; then
    unset ELASTICSEARCH_HOST
fi

if [ "${ELASTICSEARCH_USER}" == "**None**" ]; then
    unset ELASTICSEARCH_USER
fi

if [ "${ELASTICSEARCH_PASS}" == "**None**" ]; then
    unset ELASTICSEARCH_PASS
fi

echo "${HTTP_PASS}"

if [ "${HTTP_PASS}" == "**Random**" ]; then
    unset HTTP_PASS
fi

if [ "${HTTP_PASS}" != "**None**" -a ! -f /.basic_auth_configured ]; then
    /set_basic_auth.sh
fi

if [ ! -f /.influx_db_configured ]; then
    /set_influx_db.sh
fi

if [ ! -f /.elasticsearch_configured ]; then
    /set_elasticsearch.sh
fi

if [ ! -f /.dashboard_configured ]; then
    /set_dashboard.sh
fi

echo "=> Grafana for heapster version: 0.1!"
echo "=> Starting and running Nginx..."
/usr/sbin/nginx
