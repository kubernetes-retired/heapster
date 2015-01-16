#!/bin/bash

set -e

echo "${HTTP_PASS}"

if [ "${HTTP_PASS}" == "**Random**" ]; then
    unset HTTP_PASS
fi

if [ ! -f /.basic_auth_configured ]; then
    /set_basic_auth.sh
fi

if [ ! -f /.influx_db_configured ]; then
    /set_influx_db.sh
fi

if [ ! -f /.dashboard_configured ]; then
    /set_dashboard.sh
fi

echo "=> Grafana for heapster version: 0.2!"
echo "=> Starting and running Nginx..."
/usr/sbin/nginx
