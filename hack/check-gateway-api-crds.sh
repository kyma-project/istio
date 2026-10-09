#!/usr/bin/env bash
# Fails when the managed Gateway API CRDs are older than the module version in go.mod.
#
# Usage:  ./hack/check-gateway-api-crds.sh
#         make check-gateway-api-crds

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

DEST_DIR="${REPO_ROOT}/internal/reconciliations/istioresources/gateway_api_crds"

MODULE="sigs.k8s.io/gateway-api"
VERSION="$(cd "${REPO_ROOT}" && go list -m -f '{{.Version}}' "${MODULE}")"

GOMODCACHE="$(go env GOMODCACHE)"
MODULE_DIR="${GOMODCACHE}/${MODULE}@${VERSION}"

if [[ ! -d "${MODULE_DIR}" ]]; then
  (cd "${REPO_ROOT}" && go mod download "${MODULE}@${VERSION}")
fi

SRC_DIR="${MODULE_DIR}/config/crd/experimental"
CANONICAL="gateway.networking.k8s.io_gateways.yaml"

MANAGED_VERSION="$(grep "bundle-version" "${DEST_DIR}/${CANONICAL}" | awk '{print $2}')"
MODULE_CRD_VERSION="$(grep "bundle-version" "${SRC_DIR}/${CANONICAL}" | awk '{print $2}')"

echo "Managed CRD version : ${MANAGED_VERSION}"
echo "Module CRD version  : ${MODULE_CRD_VERSION}"

if [[ "${MANAGED_VERSION}" == "${MODULE_CRD_VERSION}" ]]; then
  echo "OK: Gateway API CRDs are up to date (${MANAGED_VERSION})."
  exit 0
fi

OLDEST="$(printf '%s\n%s' "${MANAGED_VERSION}" "${MODULE_CRD_VERSION}" | sort -V | head -1)"

if [[ "${OLDEST}" == "${MANAGED_VERSION}" ]]; then
  echo ""
  echo "ERROR: Managed Gateway API CRDs (${MANAGED_VERSION}) are older than the module (${MODULE_CRD_VERSION})."
  echo "Run 'make update-gateway-api-crds' and commit the result."
  exit 1
fi

echo "OK: Managed CRDs (${MANAGED_VERSION}) are ahead of the module (${MODULE_CRD_VERSION}); no action needed."
