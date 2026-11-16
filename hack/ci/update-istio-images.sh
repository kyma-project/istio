#!/usr/bin/env bash

# Bump Istio images: determine the highest patch version where all public and
# FIPS images are available, then update external-images.yaml and fips-images.yaml.
# Public images are expected to use the -distroless tag suffix.

set -euo pipefail

# shellcheck source=hack/ci/common.sh
source "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/common.sh"

require_tools curl jq yq
require_vars REGCERT_JSON

ISTIO_IMAGE_SUFFIX="-distroless"

# Returns the Istio minor version (X.Y) from the istio/pilot image in external-images.yaml.
determine_minor_version() {
  local minor

  minor="$(yq -r '.images[].source' external-images.yaml \
    | grep '^istio/pilot:' | cut -d: -f2 | cut -d. -f1-2)"

  if [ -z "${minor}" ]; then
    echo "Error: could not find istio/pilot image in external-images.yaml" >&2
    exit 1
  fi

  echo "${minor}"
}

# Fetches all istio/pilot tags for a given minor from Docker Hub and returns
# the highest patch number available with the distroless suffix.
determine_highest_possible_istio_patch() {
  local minor="$1"
  local url="https://hub.docker.com/v2/repositories/istio/pilot/tags?page_size=100&name=${minor}."
  local page code body all_tags patch

  all_tags=""
  while [ -n "${url}" ]; do
    page="$(curl -sSL -w '\n%{http_code}' "${url}" 2>/dev/null)" || true
    code="${page##*$'\n'}"
    body="${page%$'\n'*}"
    if [ "${code}" != "200" ]; then
      echo "Error: failed to fetch pilot tags for minor ${minor} (HTTP ${code})" >&2
      exit 1
    fi
    if echo "${body}" | jq -e '.message' >/dev/null 2>&1; then
      echo "Error: Docker Hub error for minor ${minor}: $(echo "${body}" | jq -r '.message')" >&2
      exit 1
    fi
    all_tags="${all_tags}"$'\n'"$(echo "${body}" | jq -r '.results[].name')"
    url="$(echo "${body}" | jq -r '.next // empty')"
  done

  patch="$(printf '%s\n' "${all_tags}" \
    | sed -nE "s/^${minor//./\\.}\\.([0-9]+)${ISTIO_IMAGE_SUFFIX}\$/\1/p" \
    | sort -n | tail -1)"

  if [ -z "${patch}" ]; then
    echo "Error: no ${ISTIO_IMAGE_SUFFIX} tags found for minor ${minor} on Docker Hub" >&2
    exit 1
  fi

  echo "${patch}"
}

# Returns 0 if the given image source (without tag) belongs to Istio.
is_istio_image() {
  local image_without_tag="$1"
  local name
  name="$(basename "${image_without_tag}")"
  echo "${name}" | grep -qE "^(istio-|ztunnel|pilot|proxyv2|install-cni)"
}

# Returns 0 if the given patch is available for all istio/* images in external-images.yaml.
is_patch_available_on_dockerhub() {
  local minor="$1" patch="$2"
  local count i src image_without_tag image_tag code body_file
  body_file="$(mktemp)"
  trap "rm -f '${body_file}'" EXIT

  count="$(yq '.images | length' external-images.yaml)"
  for i in $(seq 0 $((count - 1))); do
    src="$(yq -r ".images[${i}].source" external-images.yaml)"
    image_without_tag="$(echo "${src}" | cut -d: -f1)"
    if ! is_istio_image "${image_without_tag}"; then
      echo "Skipping non-Istio image: ${image_without_tag}"
      continue
    fi

    image_tag="${image_without_tag}:${minor}.${patch}${ISTIO_IMAGE_SUFFIX}"
    echo "Checking Docker Hub: ${image_tag}"
    code="$(curl -sSL -o "${body_file}" -w '%{http_code}' \
      "https://hub.docker.com/v2/repositories/${image_without_tag}/tags/${minor}.${patch}${ISTIO_IMAGE_SUFFIX}" \
      2>/dev/null)" || true
    if [ "${code}" = "200" ]; then
      echo "Available on Docker Hub: ${image_tag}"
    elif [ "${code}" = "404" ]; then
      echo "Not available on Docker Hub: ${image_tag}"
      return 1
    else
      echo "Unexpected response from Docker Hub for ${image_tag} (HTTP ${code}):" >&2
      cat "${body_file}" >&2
      exit 1
    fi
  done
}

# Returns 0 if the given patch is available for all FIPS images in fips-images.yaml.
is_patch_available_in_fips_registry() {
  local minor="$1" patch="$2"
  local token count i src image_without_tag code body_file
  body_file="$(mktemp)"
  trap "rm -f '${body_file}'" EXIT

  token="$(printf '_json_key:%s' "${REGCERT_JSON}" | base64 | tr -d '\n')"
  count="$(yq '.images | length' fips-images.yaml)"

  for i in $(seq 0 $((count - 1))); do
    src="$(yq -r ".images[${i}].source" fips-images.yaml)"
    image_without_tag="$(echo "${src}" | cut -d: -f1)"
    if ! is_istio_image "${image_without_tag}"; then
      echo "Skipping non-Istio image: ${image_without_tag}"
      continue
    fi

    local registry="${image_without_tag%%/*}"
    echo "Checking FIPS registry: $(basename "${image_without_tag}"):${minor}.${patch}"
    code="$(curl -sSL -o "${body_file}" -w '%{http_code}' \
      -H "Authorization: Basic ${token}" \
      "https://${registry}/v2/${image_without_tag#*/}/manifests/${minor}.${patch}" 2>/dev/null)" || true
    if [ "${code}" = "200" ]; then
      echo "Available in FIPS registry: $(basename "${image_without_tag}"):${minor}.${patch}"
    elif [ "${code}" = "404" ]; then
      echo "Not available in FIPS registry: $(basename "${image_without_tag}"):${minor}.${patch}"
      return 1
    else
      echo "Unexpected response from FIPS registry for $(basename "${image_without_tag}"):${minor}.${patch} (HTTP ${code}):" >&2
      cat "${body_file}" >&2
      exit 1
    fi
  done
}

update_external_images() {
  local minor="$1" patch="$2"
  local count i src image_without_tag new_tag

  count="$(yq '.images | length' external-images.yaml)"
  for i in $(seq 0 $((count - 1))); do
    src="$(yq -r ".images[${i}].source" external-images.yaml)"
    image_without_tag="$(echo "${src}" | cut -d: -f1)"
    if ! is_istio_image "${image_without_tag}"; then
      echo "Skipping non-Istio image: ${image_without_tag}"
      continue
    fi
    new_tag="${image_without_tag}:${minor}.${patch}${ISTIO_IMAGE_SUFFIX}"
    if [ "${src}" = "${new_tag}" ]; then
      echo "External image ${image_without_tag}: already up to date (${src})"
    else
      yq -i ".images[${i}].source = \"${new_tag}\"" external-images.yaml
      echo "External image ${image_without_tag}: updated ${src} -> ${new_tag}"
    fi
  done
}

update_fips_images() {
  local minor="$1" patch="$2"
  local count i src image_without_tag current_target new_target

  count="$(yq '.images | length' fips-images.yaml)"
  for i in $(seq 0 $((count - 1))); do
    src="$(yq -r ".images[${i}].source" fips-images.yaml)"
    image_without_tag="$(echo "${src}" | cut -d: -f1)"
    if ! is_istio_image "${image_without_tag}"; then
      echo "Skipping non-Istio image: ${image_without_tag}"
      continue
    fi
    current_target="$(yq -r ".images[${i}].target.tag" fips-images.yaml)"
    new_target="${minor}.${patch}-1"
    if [ "${src}" = "${image_without_tag}:${minor}.${patch}" ] && [ "${current_target}" = "${new_target}" ]; then
      echo "FIPS image $(basename "${image_without_tag}"): already up to date (${src})"
    else
      yq -i ".images[${i}].source = \"${image_without_tag}:${minor}.${patch}\"" fips-images.yaml
      yq -i ".images[${i}].target.tag = \"${new_target}\"" fips-images.yaml
      echo "FIPS image $(basename "${image_without_tag}"): updated source ${src} -> ${image_without_tag}:${minor}.${patch}, target ${current_target} -> ${new_target}"
    fi
  done
}

main() {
  local minor highest_patch patch best_patch

  echo "Determining minor version"
  minor="$(determine_minor_version)"
  echo "Minor version: ${minor}"

  echo "Determining highest possible Istio patch for ${minor} on Docker Hub"
  highest_patch="$(determine_highest_possible_istio_patch "${minor}")"
  echo "Highest patch on Docker Hub: ${minor}.${highest_patch}"

  best_patch=""
  for patch in $(seq "${highest_patch}" -1 0); do
    echo "Checking availability of ${minor}.${patch}"
    if is_patch_available_on_dockerhub "${minor}" "${patch}" && \
       is_patch_available_in_fips_registry "${minor}" "${patch}"; then
      best_patch="${patch}"
      break
    fi
    echo "Patch ${minor}.${patch} incomplete, trying lower"
  done

  if [ -z "${best_patch}" ]; then
    echo "No complete patch found for ${minor}; nothing to upgrade"
    return
  fi

  echo "Best available patch: ${minor}.${best_patch}"
  update_external_images "${minor}" "${best_patch}"
  update_fips_images "${minor}" "${best_patch}"
}

main
