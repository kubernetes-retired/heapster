#!/bin/bash

set -e

RELEASE="v0.4"
echo "${HTTP_PASS}"

if [ "${HTTP_PASS}" == "**Random**" ]; then
    unset HTTP_PASS
fi

if [ ! -f /.nginx_configured ]; then
    /set_nginx_conf.sh
fi

if [ ! -f /.influx_db_configured ]; then
    /set_influx_db.sh
fi

if [ ! -f /.dashboard_configured ]; then
    /set_dashboard.sh
fi

echo "=> Grafana for heapster - version: $RELEASE!"
echo "=> Starting and running Nginx..."
/usr/sbin/nginx
