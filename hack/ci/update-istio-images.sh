#!/usr/bin/env bash

set -euo pipefail

# Build map: major.minor -> newest patch from istio/pilot distroless tags,
# then update external and FIPS image manifests accordingly.

declare -A LATEST_PATCH_BY_MINOR=()

require_command() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Required command not found: ${cmd}" >&2
    exit 1
  fi
}

ensure_tools() {
  require_command curl
  require_command jq
  require_command yq
}

registry_tags() {
  local source_repo="$1"
  local registry repo_path

  registry="${source_repo%%/*}"
  repo_path="${source_repo#*/}"

  # Artifact Registry-compatible Docker V2 tags endpoint.
  curl -fsSL "https://${registry}/v2/${repo_path}/tags/list" \
    | jq -r '.tags[]?' 2>/dev/null || true
}

fetch_pilot_tags() {
  local url page
  url='https://hub.docker.com/v2/repositories/istio/pilot/tags?page_size=100'

  while [ -n "${url}" ]; do
    page="$(curl -fsSL "${url}")"
    echo "${page}" | jq -r '.results[].name'
    url="$(echo "${page}" | jq -r '.next // empty')"
  done
}

build_latest_patch_map() {
  local source minor latest_patch latest_patch_num tag patch_num
  mapfile -t pilot_tags < <(fetch_pilot_tags)

  while IFS= read -r source; do
    if [[ "${source}" =~ ^istio/.+:([0-9]+\.[0-9]+)\.[0-9]+-distroless$ ]]; then
      minor="${BASH_REMATCH[1]}"
      latest_patch=''
      latest_patch_num=-1

      for tag in "${pilot_tags[@]}"; do
        if [[ "${tag}" =~ ^${minor}\.([0-9]+)-distroless$ ]]; then
          patch_num="${BASH_REMATCH[1]}"
          if (( patch_num > latest_patch_num )); then
            latest_patch_num=${patch_num}
            latest_patch="${minor}.${patch_num}"
          fi
        fi
      done

      if [ -n "${latest_patch}" ]; then
        LATEST_PATCH_BY_MINOR["${minor}"]="${latest_patch}"
      fi
    fi
  done < <(yq -r '.images[].source' external-images.yaml)
}

update_external_images() {
  local ext_count i source image_name minor current_patch newest_patch new_source

  ext_count="$(yq '.images | length' external-images.yaml)"
  for i in $(seq 0 $((ext_count - 1))); do
    source="$(yq -r ".images[${i}].source" external-images.yaml)"
    if [[ "${source}" =~ ^(istio/[^:]+):([0-9]+\.[0-9]+)\.([0-9]+)-distroless$ ]]; then
      image_name="${BASH_REMATCH[1]}"
      minor="${BASH_REMATCH[2]}"
      current_patch="${BASH_REMATCH[2]}.${BASH_REMATCH[3]}"
      newest_patch="${LATEST_PATCH_BY_MINOR[${minor}]:-${current_patch}}"
      new_source="${image_name}:${newest_patch}-distroless"
      yq -i ".images[${i}].source = \"${new_source}\"" external-images.yaml
    fi
  done
}

update_fips_images() {
  local fips_count i source source_repo current_patch current_rev minor
  local desired_patch desired_rev desired_source_tag desired_target_tag
  local tag rev

  fips_count="$(yq '.images | length' fips-images.yaml)"
  for i in $(seq 0 $((fips_count - 1))); do
    source="$(yq -r ".images[${i}].source" fips-images.yaml)"
    if [[ ! "${source}" =~ ^(.+):([0-9]+\.[0-9]+\.[0-9]+)(-([0-9]+))?$ ]]; then
      continue
    fi

    source_repo="${BASH_REMATCH[1]}"
    current_patch="${BASH_REMATCH[2]}"
    current_rev="${BASH_REMATCH[4]:-0}"
    minor="$(echo "${current_patch}" | awk -F'.' '{print $1"."$2}')"

    desired_patch="${LATEST_PATCH_BY_MINOR[${minor}]:-${current_patch}}"
    desired_rev=0

    # If patch does not change, keep baseline at current revision.
    if [ "${desired_patch}" = "${current_patch}" ]; then
      desired_rev=${current_rev}
    fi

    mapfile -t fips_tags < <(registry_tags "${source_repo}")
    for tag in "${fips_tags[@]}"; do
      if [[ "${tag}" = "${desired_patch}" ]]; then
        continue
      elif [[ "${tag}" =~ ^${desired_patch}-([0-9]+)$ ]]; then
        rev="${BASH_REMATCH[1]}"
        if (( rev > desired_rev )); then
          desired_rev=${rev}
        fi
      fi
    done

    if (( desired_rev > 0 )); then
      desired_source_tag="${desired_patch}-${desired_rev}"
    else
      desired_source_tag="${desired_patch}"
    fi
    desired_target_tag="${desired_patch}-$((desired_rev + 1))"

    yq -i ".images[${i}].source = \"${source_repo}:${desired_source_tag}\"" fips-images.yaml
    yq -i ".images[${i}].target.tag = \"${desired_target_tag}\"" fips-images.yaml
  done
}

main() {
  ensure_tools
  build_latest_patch_map
  update_external_images
  update_fips_images
}

main
