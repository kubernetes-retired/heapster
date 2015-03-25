#!/bin/bash

set -e

RELEASE="v0.7"

if [ "$HTTP_PASS" != "**None**" ]; then
  if [ "$HTTP_PASS" = "**Random**" ]; then
    PASS_STRATEGY="random"
    HTTP_PASS="$(pwgen -s 12 1)"
  else
    PASS_STRATEGY="preset"
  fi

  echo "=> Creating basic auth for '$HTTP_USER' user with $PASS_STRATEGY password"
  echo $HTTP_PASS | htpasswd -i -c /usr/share/grafana/.htpasswd  $HTTP_USER
  echo "=> Done!"
  echo "You can now connect to Grafana with the following credential: ${HTTP_USER}:${HTTP_PASS}"
fi

echo "=> Configuring InfluxDB"
sed -i \
    -e "s/@INFLUXDB_METRICS_URL@/${INFLUXDB_METRICS_URL////\\/}/g" \
    -e "s/@INFLUXDB_GRAFANA_URL@/${INFLUXDB_GRAFANA_URL////\\/}/g" \
    -e "s/@INFLUXDB_USER@/${INFLUXDB_USER}/g" \
    -e "s/@INFLUXDB_PASS@/${INFLUXDB_PASS////\\/}/g" \
    /usr/share/grafana/config.js
echo "=> InfluxDB has been configured as follows:"
echo "   InfluxDB Metrics URL: ${INFLUXDB_METRICS_URL}"
echo "   InfluxDB Grafana URL: ${INFLUXDB_GRAFANA_URL}"
echo "   InfluxDB USERNAME: ${INFLUXDB_USER}"
echo "   InfluxDB PASSWORD: ${INFLUXDB_PASS}"
echo "   ** Please check your environment variables if you find something is misconfigured. **"
echo "=> Done!"

if [ -z "$KUBERNETES_SERVICE_HOST" ]; then
  DASHBOARD='/dashboard/file/default.json'
else
  DASHBOARD='/dashboard/file/kubernetes.json'
fi

echo "=>Setting default dashboard to $DASHBOARD"
sed -i "s/@DASHBOARD@/${DASHBOARD////\\/}/g" /usr/share/grafana/config.js
echo "=>Done"

echo "=> Grafana for heapster - version: $RELEASE!"
echo "=> Starting and running Apache..."
exec /usr/sbin/apachectl -D FOREGROUND
