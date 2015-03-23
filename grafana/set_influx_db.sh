#!/bin/bash
set -e

if [ -f /.influx_db_configured ]; then
    echo "=> InfluxDB has been configured!"
    exit 0
fi

if [ "$INFLUXDB_HOST" = "localhost" ]; then
  INFLUXDB_HOST='"+window.location.hostname+"'
elif [ "$INFLUXDB_HOST" = "**ChangeMe**" ]; then
    echo "=> No address of InfluxDB is specified!"
    echo "=> Program terminated!"
    exit 1
fi

if [ "${INFLUXDB_PORT}" = "**ChangeMe**" ]; then
    echo "=> No PORT of InfluxDB is specified!"
    echo "=> Program terminated!"
    exit 1
fi

url="${INFLUXDB_PROTO}://$INFLUXDB_HOST"
if [ -z "${KUBERNETES_API_PORT}" ]; then
  url="$url/api/v1beta1/proxy/services/monitoring-influxdb/db"
else
  url="$url:$KUBERNETES_API_PORT/api/v1beta1/proxy/services/monitoring-influxdb/db"
fi

escaped_url=${url////\\/}

echo "=> Configuring InfluxDB"
sed -i -e "s/<--URL-->/${escaped_url}/g" \
    -e "s/<--DB_NAME-->/${INFLUXDB_NAME}/g" \
    -e "s/<--GRAFANA_DB_NAME-->/${GRAFANA_DB_NAME}/g" \
    -e "s/<--USER-->/${INFLUXDB_USER}/g" \
    -e "s/<--PASS-->/${INFLUXDB_PASS////\\/}/g" \
    /app/config.js
touch /.influx_db_configured
echo "=> InfluxDB has been configured as follows:"
echo "   InfluxDB URL:     ${url}"
echo "   Grafana DB NAME:  ${GRAFANA_DB_NAME}"
echo "   InfluxDB USERNAME: ${INFLUXDB_USER}"
echo "   InfluxDB PASSWORD: ${INFLUXDB_PASS}"
echo "   ** Please check your environment variables if you find something is misconfigured. **"
echo "=> Done!"
