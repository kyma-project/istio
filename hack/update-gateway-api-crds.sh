#!/usr/bin/env bash
# Syncs Gateway API CRDs from the sigs.k8s.io/gateway-api module version in go.mod.
# Usage: make update-gateway-api-crds

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

MODULE="sigs.k8s.io/gateway-api"
VERSION="$(cd "${REPO_ROOT}" && go list -m -f '{{.Version}}' "${MODULE}")"
MODULE_DIR="$(go env GOMODCACHE)/${MODULE}@${VERSION}"
SRC_DIR="${MODULE_DIR}/config/crd/experimental"
DEST_DIR="${REPO_ROOT}/internal/reconciliations/istioresources/gateway_api_crds"
GO_FILE="${REPO_ROOT}/internal/reconciliations/istioresources/gateway_api_crds.go"

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

not_embedded=()
for f in "${DEST_DIR}"/gateway.networking.k8s.io_*.yaml; do
  grep -qF "$(basename "${f}")" "${GO_FILE}" || not_embedded+=("$(basename "${f}")")
done

if [[ ${#not_embedded[@]} -eq 0 ]]; then
  echo "Embed check: all CRDs are registered in ${GO_FILE##"${REPO_ROOT}/"}"
else
  echo ""
  echo "⚠️  WARNING: the following CRD(s) are NOT registered in ${GO_FILE##"${REPO_ROOT}/"}"
  echo "   and will be IGNORED by the operator at runtime:"
  for name in "${not_embedded[@]}"; do echo "     - ${name}"; done
  echo ""
  echo "   Add a //go:embed directive and an entry in gatewayAPICRDManifests for each file above."
fi
