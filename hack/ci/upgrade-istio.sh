#!/usr/bin/env bash

set -euo pipefail

# Perform the Istio upgrade after the external and FIPS images have been
# synced by update-istio-images.sh (and its PR merged). This script:
#   1. reads the target Istio version from the in-repo synced manifests
#      (external-images.yaml, fips-images.yaml),
#   2. repoints config/manager/env-images.yaml and the operator manifests at it,
#   3. bumps the istio.io/istio Go dependency (go get -u -t + go mod tidy),
#   4. appends an "Istio Updated to Version X.Y.Z" section to the current
#      release-notes draft, or creates the next-version note file if the current
#      one is already released (has a matching git tag).
#
# It never queries a registry: the target version is whatever the image-update
# automation already chose. If env-images.yaml is already at (or newer than)
# that version, the script is a no-op.

ENV_IMAGES="config/manager/env-images.yaml"
EXTERNAL_IMAGES="external-images.yaml"
FIPS_IMAGES="fips-images.yaml"
RELEASE_NOTES_DIR="docs/release-notes"
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

set_output() { [ -n "${GITHUB_OUTPUT:-}" ] && echo "$1=$2" >> "${GITHUB_OUTPUT}" || true; }

ensure_tools() {
  local cmd
  for cmd in yq go git sed; do
    command -v "${cmd}" >/dev/null 2>&1 || { echo "Required command not found: ${cmd}" >&2; exit 1; }
  done
}

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

# Highest docs/release-notes/X.Y.Z.md whose version has no matching git tag:
# the accumulating draft for the next release. Empty if all are released.
draft_release_note() {
  local f v
  while IFS= read -r f; do
    v="$(basename "${f}" .md)"
    [[ "${v}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || continue
    git rev-parse -q --verify "refs/tags/${v}" >/dev/null 2>&1 && continue
    echo "${f}"; return 0
  done < <(printf '%s\n' "${RELEASE_NOTES_DIR}"/*.md | sort -Vr)
  return 1
}

# Path of the next release-note file to create when the current one is released.
# main         -> next minor:  highest X.Y.Z tag -> X.(Y+1).0.md
# release-X.Y  -> next patch:   highest X.Y.Z tag -> X.Y.(Z+1).md
next_release_note_path() {
  local base minor latest maj min pat
  base="${BASE_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}"
  if [[ "${base}" == release-* ]]; then
    minor="${base#release-}" # X.Y
    latest="$(git tag --list | grep -E "^${minor//./\\.}\.[0-9]+$" | sort -V | tail -1)"
    if [ -n "${latest}" ]; then
      pat="${latest##*.}"
      echo "${RELEASE_NOTES_DIR}/${minor}.$((pat + 1)).md"
    else
      echo "${RELEASE_NOTES_DIR}/${minor}.0.md"
    fi
  else
    latest="$(git tag --list | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1)"
    maj="${latest%%.*}"; min="${latest#*.}"; min="${min%%.*}"
    echo "${RELEASE_NOTES_DIR}/${maj}.$((min + 1)).0.md"
  fi
}

main() {
  ensure_tools

  local current minor target highest
  current="$(env_value pilot | sed -nE 's#.*:([0-9]+\.[0-9]+\.[0-9]+)-distroless$#\1#p')"
  [ -n "${current}" ] || { echo "Could not read current Istio version from ${ENV_IMAGES}" >&2; exit 1; }
  minor="${current%.*}"

  target="$(target_distroless_version "${minor}")"
  [ -n "${target}" ] || { echo "No distroless target for minor ${minor} in ${EXTERNAL_IMAGES}" >&2; exit 1; }

  # Never downgrade: no-op unless target is strictly newer than current.
  highest="$(printf '%s\n%s\n' "${current}" "${target}" | sort -V | tail -1)"
  if [ "${target}" = "${current}" ] || [ "${highest}" = "${current}" ]; then
    echo "env-images.yaml at ${current}, target ${target} is not newer; nothing to upgrade."
    set_output has_changes false
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
    case "${ftag}" in
      "${target}-"*) : ;;
      *)
        echo "FIPS image ${FIPS_VAR_TO_IMG[${fv}]} at '${ftag:-<none>}' has not caught up to distroless target ${target}; skipping until synced."
        set_output has_changes false
        exit 0
        ;;
    esac
  done

  # Distroless images: keep the registry path, swap the tag.
  local v cur repo tag
  for v in "${DISTROLESS_VARS[@]}"; do
    cur="$(env_value "${v}")"; repo="${cur%:*}"
    set_env_value "${v}" "${repo}:${target}-distroless"
  done

  # FIPS images: tag comes verbatim from fips-images.yaml target.tag.
  for v in "${!FIPS_VAR_TO_IMG[@]}"; do
    cur="$(env_value "${v}")"; repo="${cur%:*}"
    tag="$(fips_target_tag "${FIPS_VAR_TO_IMG[${v}]}" "${minor}")"
    [ -n "${tag}" ] || { echo "No FIPS tag for ${v} (${FIPS_VAR_TO_IMG[${v}]}) minor ${minor}" >&2; exit 1; }
    set_env_value "${v}" "${repo}:${tag}"
  done

  # Operator manifests (regular + light; the experimental profile reuses these).
  # Anchored sed on the single top-level `tag:` line, to avoid yq reflowing the
  # whole hand-formatted manifest.
  local f
  for f in "${OPERATOR_YAMLS[@]}"; do
    sed -i.bak -E "s#^(  tag: \").*(\")#\1${target}-distroless\2#" "${f}"
    rm -f "${f}.bak"
  done

  # Go dependency: mirror the manual upgrade (go get -u -t istio@version + tidy).
  echo "Bumping istio.io/istio to ${target}"
  go get -u -t "istio.io/istio@${target}"
  go mod tidy

  # Release note: append to the current draft, or create the next-version file.
  local note heading announce
  heading="## Istio Updated to Version ${target}"
  announce="https://istio.io/latest/news/releases/${minor}.x/announcing-${target}/"
  if note="$(draft_release_note)"; then
    if grep -qF "${heading}" "${note}"; then
      echo "Release note already mentions ${target} in ${note}"
    else
      printf '\n%s\n\nWe'\''ve updated the Istio version to %s. See [Announcing Istio %s](%s).\n' \
        "${heading}" "${target}" "${target}" "${announce}" >> "${note}"
      echo "Added Istio ${target} section to ${note}"
    fi
  else
    note="$(next_release_note_path)"
    printf '%s\n\nWe'\''ve updated the Istio version to %s. See [Announcing Istio %s](%s).\n' \
      "${heading}" "${target}" "${target}" "${announce}" > "${note}"
    echo "Created ${note} with Istio ${target} section"
  fi

  set_output has_changes true
  set_output istio_version "${target}"
  set_output announce_url "${announce}"
}

main
