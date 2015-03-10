#!/bin/bash
set -e

if [ -f /.influx_db_configured ]; then
    echo "=> InfluxDB has been configured!"
    exit 0
fi
echo "=> Configuring InfluxDB"
sed -i -e "s/<--DB_NAME-->/${INFLUXDB_NAME}/g" \
    -e "s/<--GRAFANA_DB_NAME-->/${GRAFANA_DB_NAME}/g" \
    -e "s/<--USER-->/${INFLUXDB_USER}/g" \
    -e "s/<--PASS-->/${INFLUXDB_PASS}/g" /app/config.js
touch /.influx_db_configured
echo "=> InfluxDB has been configured as follows:"
echo "   Grafana DB NAME:  ${GRAFANA_DB_NAME}"
echo "   InfluxDB USERNAME: ${INFLUXDB_USER}"
echo "   InfluxDB PASSWORD: ${INFLUXDB_PASS}"
echo "   ** Please check your environment variables if you find something is misconfigured. **"
echo "=> Done!"
