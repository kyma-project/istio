#!/usr/bin/env bash
# Syncs Gateway API CRDs from the sigs.k8s.io/gateway-api module version in go.mod.
# Usage: make update-gateway-api-crds

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

MODULE="sigs.k8s.io/gateway-api"
VERSION="$(cd "${REPO_ROOT}" && go list -m -f '{{.Version}}' "${MODULE}")"
MODULE_DIR="$(go env GOMODCACHE)/${MODULE}@${VERSION}"
SRC_DIR="${MODULE_DIR}/config/crd/standard"
DEST_DIR="${REPO_ROOT}/internal/reconciliations/istioresources/gateway_api_crds"

[[ -d "${MODULE_DIR}" ]] || (cd "${REPO_ROOT}" && go mod download "${MODULE}@${VERSION}")

echo "Syncing Gateway API CRDs → ${VERSION}"

for f in "${DEST_DIR}"/gateway.networking.k8s.io_*.yaml; do
  [[ -f "${SRC_DIR}/$(basename "${f}")" ]] || { echo "  removing orphan: $(basename "${f}")"; rm "${f}"; }
done

for src in "${SRC_DIR}"/gateway.networking.k8s.io_*.yaml; do
  cp "${src}" "${DEST_DIR}/$(basename "${src}")"
  echo "  updated: $(basename "${src}")"
done

echo "Done."
