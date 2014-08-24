#!/bin/bash

set -e

if [ -f /.elasticsearch_configured ]; then
    echo "=> Elasticsearch has been configured!"
    exit 0
fi

if [ -n "${ELASTICSEARCH_HOST}" ] && [ -n "${ELASTICSEARCH_PORT}" ]; then
    echo "=> Found Elasticsearch settings."
    ip=${ELASTICSEARCH_HOST}
    if [ "$ip" = "localhost" ]; then
	ip=$(curl "http://metadata/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip" -H "X-Google-Metadata-Request: True")
    fi
    if [ -n "${ELASTICSEARCH_USER}" ] && [ -n "${ELASTICSEARCH_PASS}" ]; then
        echo "=> Set Elasticsearch url to \"${ELASTICSEARCH_PROTO}://${ELASTICSEARCH_USER}:${ELASTICSEARCH_PASS}@$ip:${ELASTICSEARCH_PORT}\"."
        sed -i "s#.*elasticsearch.*#elasticsearch:\"${ELASTICSEARCH_PROTO}://${ELASTICSEARCH_USER}:${ELASTICSEARCH_PASS}@$ip:${ELASTICSEARCH_PORT}\",#g"  /app/config.js
    else
        echo "=> Set Elasticsearch url to \"${ELASTICSEARCH_PROTO}://$ip:${ELASTICSEARCH_PORT}\"."
        sed -i "s#.*elasticsearch.*#elasticsearch:\"${ELASTICSEARCH_PROTO}://$ip:${ELASTICSEARCH_PORT}\",#g"  /app/config.js
    fi
    echo "=> Done!"
else
    echo "=> Either address or port of Elasticsearch is not set or empty."
    echo "=> Skip setting Elasticsearch."
fi

touch /.elasticsearch_configured
