#!/bin/bash

set -e

if [ -f /.basic_auth_configured ]; then
    echo "=> Basic HTTP auth has been configured!"
    exit 0
fi

PASS=${HTTP_PASS:-$(pwgen -s 12 1)}
_word=$([ ${HTTP_PASS} ] && echo "preset" || echo "random")

echo "=> Creating basic auth for \" ${HTTP_USER}\" user with ${_word} password"
echo ${PASS} | htpasswd -i -c /app/.htpasswd  ${HTTP_USER}
echo "=> Done!"
touch /.basic_auth_configured

echo "========================================================================"
echo "You can now connect to Grafana with the following credential:"
echo ""
echo "    ${HTTP_USER}:${PASS}"
echo ""
echo "========================================================================"
