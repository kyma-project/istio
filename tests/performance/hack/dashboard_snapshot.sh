#!/usr/bin/env bash

DASHBOARD_UID=$(curl -s --request GET \
  --url "https://grafana.${KYMA_DOMAIN}/api/search?query=Istio-performance" \
  --header 'Authorization: Basic YWRtaW46YWRtaW4=' | jq '.[0].uid' | sed 's/["]//g')

DASHBOARD=$(curl -s --request GET \
  --url "https://grafana.${KYMA_DOMAIN}/api/dashboards/uid/${DASHBOARD_UID}" \
  --header 'Authorization: Basic YWRtaW46YWRtaW4=' | jq  '.dashboard')

key=$(xxd -l16 -ps /dev/urandom)
deleteKey=$(xxd -l16 -ps /dev/urandom)

set -x
SNAPSHOT_URL=$(curl -s --request POST \
  --url "https://grafana.${KYMA_DOMAIN}/api/snapshots" \
  -d '{"dashboard": '"${DASHBOARD}"', "name": "Istio-performance","expires": 0, "external": true, "key": "'"${key}"'", "deleteKey": "'"${deleteKey}"'"}' \
  --header 'Authorization: Basic YWRtaW46YWRtaW4=' \
  --header 'Content-Type: application/json' | jq '.url')

echo $SNAPSHOT_URL