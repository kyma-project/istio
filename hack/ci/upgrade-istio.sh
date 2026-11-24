#!/usr/bin/env bash

# Perform the Istio upgrade after the external and FIPS images have been
# synced by update-istio-images.sh (and its PR merged). This script:
#   1. reads the target Istio version from the in-repo synced manifests
#      (external-images.yaml, fips-images.yaml),
#   2. repoints config/manager/env-images.yaml, the operator manifests, and
#      component-config.yaml at it,
#   3. bumps the istio.io/istio Go dependency (go get -u -t + go mod tidy),
#   4. writes the new Istio version and PR body to the caller-supplied output files.
#
# Usage: upgrade-istio.sh <istio-version-file> <pr-body-file>
#
# Exit 0: script completed (files written only if an upgrade was performed).
# Exit non-zero: unexpected error.
#
# It never queries a registry: the target version is whatever the image-update
# automation already chose. If env-images.yaml is already at (or newer than)
# that version, the script is a no-op.

set -euo pipefail

script_dir="$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")"
# shellcheck source=hack/ci/common.sh
source "${script_dir}/common.sh"

require_bash 4
require_tools yq go git sed envsubst
require_positional istio_version_file "$1"
require_positional pr_body_file "$2"

# All relative paths in this script (env-images.yaml, go.mod, etc.) are
# relative to the repo root, so cd there unconditionally.
cd "${script_dir}/../.."

ENV_IMAGES="config/manager/env-images.yaml"
EXTERNAL_IMAGES="external-images.yaml"
FIPS_IMAGES="fips-images.yaml"
COMPONENT_CONFIG="component-config.yaml"
OPERATOR_YAMLS=(
  "internal/istiooperator/istio-operator.yaml"
  "internal/istiooperator/istio-operator-light.yaml"
)

# env-images.yaml FIPS var -> fips-images.yaml target.name suffix.
declare -A FIPS_VAR_TO_IMG=(
  [install-cni-fips]="istio-install-cni-fips"
  [proxyv2-fips]="istio-proxy-fips"
  [pilot-fips]="istio-pilot-fips"
  [ztunnel-fips]="ztunnel-fips"
)
DISTROLESS_VARS=(install-cni proxyv2 pilot ztunnel)

# Value of a named env var in the manager container of env-images.yaml.
env_value() {
  NAME="$1" yq '.spec.template.spec.containers[]
    | select(.name == "manager") | .env[]
    | select(.name == strenv(NAME)) | .value' "${ENV_IMAGES}"
}

set_env_value() {
  NAME="$1" VALUE="$2" yq -i '(.spec.template.spec.containers[]
    | select(.name == "manager") | .env[]
    | select(.name == strenv(NAME))).value = strenv(VALUE)' "${ENV_IMAGES}"
}

# Highest istio/<name>:<minor>.<patch>-distroless patch in external-images.yaml.
target_distroless_version() {
  local minor="$1"
  yq '.images[].source' "${EXTERNAL_IMAGES}" \
    | sed -nE "s#^istio/pilot:(${minor//./\\.}\.[0-9]+)-distroless\$#\1#p" \
    | sort -V | tail -1
}

# Highest target.tag for a given fips target.name suffix, within a minor.
fips_target_tag() {
  local img="$1" minor="$2"
  IMG="external/istio/${img}" yq '.images[]
    | select(.target.name == strenv(IMG)) | .target.tag' "${FIPS_IMAGES}" \
    | sed -nE "s#^(${minor//./\\.}\.[0-9]+-[0-9]+)\$#\1#p" \
    | sort -V | tail -1
}

# component-config.yaml lists the images shipped on the branch for main-branch
# security scanning as a flat list. Its non-manager entries carry the exact
# registry path and tag of their env-images.yaml counterparts, so repoint each
# one to the value this upgrade just wrote. Anchored sed keyed on the unique
# repo path, matching the operator-manifest approach, to avoid yq reflowing the
# hand-formatted file. The istio-manager:main entry has no env-images
# counterpart and is left untouched; images absent from component-config.yaml
# (e.g. ztunnel) match no line and are skipped.
update_component_config() {
  [ -f "${COMPONENT_CONFIG}" ] || return 0
  local var val repo
  for var in "${DISTROLESS_VARS[@]}" "${!FIPS_VAR_TO_IMG[@]}"; do
    val="$(env_value "${var}")"
    repo="${val%:*}"
    sed -i.bak -E "s#^(  - ${repo//./\\.}:).*#\1${val##*:}#" "${COMPONENT_CONFIG}"
    rm -f "${COMPONENT_CONFIG}.bak"
  done
}

main() {
  local current minor target highest
  current="$(env_value pilot | sed -nE 's#.*:([0-9]+\.[0-9]+\.[0-9]+)-distroless$#\1#p')"
  [ -n "${current}" ] || { echo "Could not read current Istio version from ${ENV_IMAGES}" >&2; exit 1; }
  minor="${current%.*}"
  echo "Current version: ${current}, minor: ${minor}"

  target="$(target_distroless_version "${minor}")"
  [ -n "${target}" ] || { echo "No distroless target for minor ${minor} in ${EXTERNAL_IMAGES}" >&2; exit 1; }
  echo "Target version from ${EXTERNAL_IMAGES}: ${target}"

  # Never downgrade: no-op unless target is strictly newer than current.
  highest="$(printf '%s\n%s\n' "${current}" "${target}" | sort -V | tail -1)"
  echo "Highest of current/target: ${highest}"
  if [ "${target}" = "${current}" ]; then
    echo "Current version ${current} matches target; nothing to upgrade."
    exit 0
  elif [ "${highest}" = "${current}" ]; then
    echo "Current version ${current} is ahead of target ${target}; nothing to upgrade."
    exit 0
  fi
  echo "Upgrading Istio ${current} -> ${target} (minor ${minor})"

  # FIPS images are mirrored separately and may lag the distroless patch by a
  # sync cycle. Adopt a version only once its FIPS tags have caught up to the
  # same patch, else env-images.yaml would pin mismatched distroless/FIPS Istio
  # versions. Not-yet-synced is a clean no-op, checked before any mutation.
  local fv ftag
  for fv in "${!FIPS_VAR_TO_IMG[@]}"; do
    ftag="$(fips_target_tag "${FIPS_VAR_TO_IMG[${fv}]}" "${minor}")"
    echo "FIPS check: ${fv} (${FIPS_VAR_TO_IMG[${fv}]}) -> tag: ${ftag:-<none>}"
    case "${ftag}" in
      "${target}-"*) : ;;
      *)
        echo "FIPS image ${FIPS_VAR_TO_IMG[${fv}]} at '${ftag:-<none>}' has not caught up to distroless target ${target}; skipping until synced."
        exit 0
        ;;
    esac
  done
  echo "All FIPS images are in sync with target ${target}"

  # Distroless images: keep the registry path, swap the tag.
  local v cur repo tag
  for v in "${DISTROLESS_VARS[@]}"; do
    cur="$(env_value "${v}")"; repo="${cur%:*}"
    set_env_value "${v}" "${repo}:${target}-distroless"
    echo "Updated distroless ${v}: ${cur} -> ${repo}:${target}-distroless"
  done

  # FIPS images: tag comes verbatim from fips-images.yaml target.tag.
  for v in "${!FIPS_VAR_TO_IMG[@]}"; do
    cur="$(env_value "${v}")"; repo="${cur%:*}"
    tag="$(fips_target_tag "${FIPS_VAR_TO_IMG[${v}]}" "${minor}")"
    [ -n "${tag}" ] || { echo "No FIPS tag for ${v} (${FIPS_VAR_TO_IMG[${v}]}) minor ${minor}" >&2; exit 1; }
    set_env_value "${v}" "${repo}:${tag}"
    echo "Updated FIPS ${v}: ${cur} -> ${repo}:${tag}"
  done

  # Operator manifests (regular + light; the experimental profile reuses these).
  # Anchored sed on the single top-level `tag:` line, to avoid yq reflowing the
  # whole hand-formatted manifest.
  local f
  for f in "${OPERATOR_YAMLS[@]}"; do
    sed -i.bak -E "s#^(  tag: \").*(\")#\1${target}-distroless\2#" "${f}"
    rm -f "${f}.bak"
  done

  # component-config.yaml tracks the same images for security scanning.
  update_component_config

  # Go dependency: mirror the manual upgrade (go get -u -t istio@version + tidy).
  echo "Bumping istio.io/istio to ${target}"
  go get -u -t "istio.io/istio@${target}"
  go mod tidy

  # Gateway API CRDs sync with Istio
  echo "Syncing Gateway API CRDs with Istio"
  "${script_dir}/../update-gateway-api-crds.sh"

  # Write output files for the caller.
  local announce
  announce="https://istio.io/latest/news/releases/${minor}.x/announcing-${target}/"

  echo "${target}" > "${istio_version_file}"

  ISTIO_VERSION="${target}" ANNOUNCE_URL="${announce}" \
    envsubst '$ISTIO_VERSION $ANNOUNCE_URL' \
    < "${script_dir}/istio-upgrade-pr-body.md.tmpl" \
    > "${pr_body_file}"
}

main
