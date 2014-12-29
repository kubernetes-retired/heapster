#!/bin/bash
set -e

if [ -f /.influx_db_configured ]; then
    echo "=> InfluxDB has been configured!"
    exit 0
fi

dbIP=${INFLUXDB_HOST}
if [ "$dbIP" = "localhost" ]; then
  dbIP='"+window.location.hostname+"'
elif [ "$dbIP" = "**ChangeMe**" ]; then
    echo "=> No address of InfluxDB is specified!"
    echo "=> Program terminated!"
    exit 1
fi

if [ "${INFLUXDB_PORT}" = "**ChangeMe**" ]; then
    echo "=> No PORT of InfluxDB is specified!"
    echo "=> Program terminated!"
    exit 1
fi

echo "=> Configuring InfluxDB"
sed -i -e "s/<--PROTO-->/${INFLUXDB_PROTO}/g" \
    -e "s/<--ADDR-->/$dbIP/g" \
    -e "s/<--PORT-->/${INFLUXDB_PORT}/g" \
    -e "s/<--DB_NAME-->/${INFLUXDB_NAME}/g" \
    -e "s/<--GRAFANA_DB_NAME-->/${GRAFANA_DB_NAME}/g" \
    -e "s/<--USER-->/${INFLUXDB_USER}/g" \
    -e "s/<--PASS-->/${INFLUXDB_PASS}/g" /app/config.js
touch /.influx_db_configured
echo "=> InfluxDB has been configured as follows:"
echo "   InfluxDB ADDRESS:  $dbIP"
echo "   InfluxDB PORT:     ${INFLUXDB_PORT}"
echo "   InfluxDB DB NAME:  ${INFLUXDB_NAME}"
echo "   Grafana DB NAME:  ${GRAFANA_DB_NAME}"
echo "   InfluxDB USERNAME: ${INFLUXDB_USER}"
echo "   InfluxDB PASSWORD: ${INFLUXDB_PASS}"
echo "   ** Please check your environment variables if you find something is misconfigured. **"
echo "=> Done!"
