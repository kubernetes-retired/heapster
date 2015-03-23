#!/bin/bash
set -e

if [ -f /.influx_db_configured ]; then
    echo "=> InfluxDB has been configured!"
    exit 0
fi

echo "=> Configuring InfluxDB"
sed -i \
    -e "s/<--INFLUXDB_METRICS_URL-->/${INFLUXDB_METRICS_URL////\\/}/g" \
    -e "s/<--INFLUXDB_GRAFANA_URL-->/${INFLUXDB_GRAFANA_URL////\\/}/g" \
    -e "s/<--INFLUXDB_USER-->/${INFLUXDB_USER}/g" \
    -e "s/<--INFLUXDB_PASS-->/${INFLUXDB_PASS////\\/}/g" \
    /app/config.js
touch /.influx_db_configured
echo "=> InfluxDB has been configured as follows:"
echo "   InfluxDB Metrics URL: ${INFLUXDB_METRICS_URL}"
echo "   InfluxDB Grafana URL: ${INFLUXDB_GRAFANA_URL}"
echo "   InfluxDB USERNAME: ${INFLUXDB_USER}"
echo "   InfluxDB PASSWORD: ${INFLUXDB_PASS}"
echo "   ** Please check your environment variables if you find something is misconfigured. **"
echo "=> Done!"
