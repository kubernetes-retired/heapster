#!/bin/bash

set -e

if [ -f /.nginx_configured ]; then
    echo "=> Nginx is already configured!"
    exit 0
fi

if [ $HTTP_PASS == "**None**" ]; then
    echo "=> Setting up Grafana without any auth"
    sed -i "s/<--AuthInfo-->//g" /etc/nginx/sites-enabled/default
    echo "=> Done!"
else
    PASS=${HTTP_PASS:-$(pwgen -s 12 1)}
    _word=$([ ${HTTP_PASS} ] && echo "preset" || echo "random")
    NGINX_CONFIG='auth_basic \"Restricted\";auth_basic_user_file \/app\/\.htpasswd;'
    echo "=> Creating basic auth for \" ${HTTP_USER}\" user with ${_word} password"
    echo ${PASS} | htpasswd -i -c /app/.htpasswd  ${HTTP_USER}
    sed -i "s/<--AuthInfo-->/$NGINX_CONFIG/g" /etc/nginx/sites-enabled/default
fi

# InfluxDB proxy configuration
echo "=> Setting up InfluxDB proxy"

if [ -n "$INFLUXDB_PROTO" -o "$INFLUXDB_PROTO" == "**ChangeMe**" ]; then
    export INFLUXDB_PROTO="http"
fi
sed -i "s/<--DBPROTO-->/$INFLUXDB_PROTO/g" /etc/nginx/sites-enabled/default

if [ -n "$INFLUXDB_HOST" -o "$INFLUXDB_HOST" == "**ChangeMe**" ]; then
    export INFLUXDB_HOST="localhost"
fi
sed -i "s/<--DBHOST-->/$INFLUXDB_HOST/g" /etc/nginx/sites-enabled/default

if [ -n  "$INFLUXDB_PORT" -o "$INFLUXDB_PORT" == "**ChangeMe**" ]; then
    export INFLUXDB_PORT="8086"
fi
sed -i "s/<--DBPORT-->/$INFLUXDB_PORT/g" /etc/nginx/sites-enabled/default
echo "=> Done!"

touch /.nginx_configured

echo "========================================================================"
echo "You can now connect to Grafana with the following credential:"
echo ""
echo "    ${HTTP_USER}:${PASS}"
echo ""
echo "InfluxDB is now proxied at:"
echo ""
echo "     $INFLUXDB_PROTO://$INFLUXDB_HOST:$INFLUXDB_PORT"
echo ""
echo "========================================================================"
