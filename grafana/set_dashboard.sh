#!/bin/sh

set -e

DASHBOARD='\/dashboard\/file\/default.json'

if [ ! -z $KUBERNETES_SERVICE_HOST ]; then
    DASHBOARD='\/dashboard\/file\/kubernetes.json'
fi

echo "=>Setting default dashboard to $DASHBOARD"
sed -i "s/<--DASHBOARD-->/$DASHBOARD/g" /app/config.js
touch /.dashboard_configured
echo "=>Done"