#!/usr/bin/env bash

# Verify that all Istio images in external-images.yaml and fips-images.yaml
# reference the same patch version (X.Y.Z).

set -euo pipefail

# shellcheck source=hack/ci/common.sh
source "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/common.sh"

require_tools yq

is_istio_image() {
  local image_without_tag="$1"
  local name
  name="$(basename "${image_without_tag}")"
  echo "${name}" | grep -qE "^(istio-|ztunnel|pilot|proxyv2|install-cni)"
}

collect_versions() {
  local file="$1"
  local count i src image_without_tag tag version
  count="$(yq '.images | length' "${file}")"
  for i in $(seq 0 $((count - 1))); do
    src="$(yq -r ".images[${i}].source" "${file}")"
    image_without_tag="${src%%:*}"
    if ! is_istio_image "${image_without_tag}"; then
      continue
    fi
    tag="${src#*:}"
    version="${tag%%-*}"  # strip -distroless or -1 suffix
    echo "${version}"
  done
}

main() {
  local all_versions unique_versions version_count

  echo "Collecting Istio versions from external-images.yaml and fips-images.yaml"
  all_versions="$(collect_versions external-images.yaml; collect_versions fips-images.yaml)"

  echo "Found versions:"
  echo "${all_versions}"

  unique_versions="$(echo "${all_versions}" | sort -u)"
  version_count="$(echo "${unique_versions}" | wc -l | tr -d ' ')"

  if [ "${version_count}" -ne 1 ]; then
    echo "Inconsistent Istio patch versions across image files:" >&2
    echo "${unique_versions}" >&2
    exit 1
  fi

  echo "All Istio images are consistent at version ${unique_versions}"
}

main
