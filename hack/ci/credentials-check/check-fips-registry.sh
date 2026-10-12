#!/usr/bin/env bash

# Verifies FIPS_REGISTRY_SA by authenticating against the FIPS image registry
# (europe-docker.pkg.dev/kyma-project/restricted-prod) using the service account JSON.

set -eo pipefail
script_dir="$(dirname "$(readlink -f "$0")")"
# shellcheck source=../common.sh
source "${script_dir}/../common.sh"

require_vars FIPS_REGISTRY_SA

start_group "Validate service account JSON"
if ! echo "${FIPS_REGISTRY_SA}" | jq empty 2>/dev/null; then
  >&2 echo "FAIL: FIPS_REGISTRY_SA is not valid JSON"
  exit 1
fi

CLIENT_EMAIL=$(echo "${FIPS_REGISTRY_SA}" | jq -r '.client_email // empty')
if [ -z "${CLIENT_EMAIL}" ]; then
  >&2 echo "FAIL: FIPS_REGISTRY_SA has no client_email field"
  exit 1
fi
end_group

start_group "Authenticate against FIPS registry"
TOKEN=$(printf '_json_key:%s' "${FIPS_REGISTRY_SA}" | base64 | tr -d '\n')
HTTP_STATUS=$(curl --silent --output /dev/null --write-out "%{http_code}" --location \
  --header "Authorization: Basic ${TOKEN}" \
  "https://europe-docker.pkg.dev/v2/kyma-project/restricted-prod/token?service=europe-docker.pkg.dev")
echo "HTTP status: ${HTTP_STATUS}"
if [ "${HTTP_STATUS}" -ne 200 ]; then
  >&2 echo "FAIL: registry authentication returned HTTP ${HTTP_STATUS} — service account may be expired or lack access"
  exit 1
fi
end_group

echo "PASS: FIPS registry credentials are valid"
