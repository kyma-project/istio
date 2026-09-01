#!/usr/bin/env bash

set -euo pipefail

# Bump Istio images: pick the newest istio/pilot patch per minor from Docker Hub,
# but only up to the newest patch actually available for every FIPS image, then
# update external-images.yaml and fips-images.yaml.

declare -A LATEST_PATCH_BY_MINOR=()

ensure_tools() {
  local cmd
  for cmd in curl jq yq; do
    command -v "${cmd}" >/dev/null 2>&1 || { echo "Required command not found: ${cmd}" >&2; exit 1; }
  done
}

registry_auth_token() {
  [ -n "${REGCERT_JSON:-}" ] || return 1
  echo "${REGCERT_JSON}" | jq -e '.client_email' >/dev/null 2>&1 || {
    echo "Warning: no client_email in REGCERT_JSON" >&2; return 1; }
  printf '_json_key:%s' "${REGCERT_JSON}" | base64 | tr -d '\n'
}

registry_tags() {
  local registry="${1%%/*}" path="${1#*/}" opts=(-sSL -w '%{http_code}') token resp code url
  url="https://${registry}/v2/${path}/tags/list"
  if [ -n "${REGCERT_JSON:-}" ]; then
    token="$(registry_auth_token 2>/dev/null)" || true
    [ -n "${token}" ] && opts+=(-H "Authorization: Basic ${token}")
  fi
  resp="$(curl "${opts[@]}" "${url}" 2>/dev/null)" || true
  code="${resp: -3}"
  if [ "${code}" != "200" ]; then
    echo "Warning: failed to fetch tags (HTTP ${code}) from ${url}" >&2
    return 1
  fi
  echo "${resp%???}" | jq -r '.tags[]?' 2>/dev/null || true
}

fetch_pilot_tags_for_minor() {
  local minor="$1"
  local url="https://hub.docker.com/v2/repositories/istio/pilot/tags?page_size=100&name=${minor}."
  local page code body
  while [ -n "${url}" ]; do
    page="$(curl -sSL -w '\n%{http_code}' "${url}" 2>/dev/null)" || true
    code="${page##*$'\n'}"
    body="${page%$'\n'*}"
    if [ "${code}" != "200" ]; then
      echo "Warning: failed to fetch pilot tags for minor ${minor} (HTTP ${code}) from ${url}" >&2
      return 1
    fi
    if echo "${body}" | jq -e '.message' >/dev/null 2>&1; then
      echo "Warning: Docker Hub error for minor ${minor}: $(echo "${body}" | jq -r '.message')" >&2
      return 1
    fi
    echo "${body}" | jq -r '.results[].name'
    url="$(echo "${body}" | jq -r '.next // empty')"
  done
}

# Highest N among stdin tags matching "<prefix>.<N><suffix>".
highest_num() { sed -nE "s/^${1//./\\.}\.([0-9]+)${2}\$/\1/p" | sort -n | tail -1; }
# Highest N among stdin tags matching "<version>-<N>".
highest_rev() { sed -nE "s/^${1//./\\.}-([0-9]+)\$/\1/p" | sort -n | tail -1; }

img_count() { yq '.images | length' "$1"; }

build_latest_patch_map() {
  local minor patch tags
  for minor in $(yq -r '.images[].source' external-images.yaml \
      | sed -nE 's#^istio/[^:]+:([0-9]+\.[0-9]+)\.[0-9]+-distroless$#\1#p' | sort -u); do
    tags="$(fetch_pilot_tags_for_minor "${minor}")" || continue
    patch="$(printf '%s\n' "${tags}" | highest_num "${minor}" '-distroless')"
    [ -n "${patch}" ] && LATEST_PATCH_BY_MINOR["${minor}"]="${minor}.${patch}"
  done
}

adjust_patch_map_with_fips() {
  local src repo minor p best dockerhub
  local -A total=() count=() current=()

  while IFS= read -r src; do
    [[ "${src}" =~ ^(.+):([0-9]+\.[0-9]+)\.([0-9]+)(-[0-9]+)?$ ]] || continue
    repo="${BASH_REMATCH[1]}"; minor="${BASH_REMATCH[2]}"
    current["${minor}"]="${minor}.${BASH_REMATCH[3]}"
    [ -n "${LATEST_PATCH_BY_MINOR[${minor}]:-}" ] || continue
    total["${minor}"]=$(( ${total["${minor}"]:-0} + 1 ))
    while read -r p; do
      count["${minor}:${p}"]=$(( ${count["${minor}:${p}"]:-0} + 1 ))
    done < <(registry_tags "${repo}" | sed -nE "s/^${minor//./\\.}\.([0-9]+)(-[0-9]+)?\$/\1/p" | sort -un)
  done < <(yq -r '.images[].source' fips-images.yaml)

  for minor in "${!total[@]}"; do
    dockerhub="${LATEST_PATCH_BY_MINOR[${minor}]}"
    best=''
    for ((p = 0; p <= ${dockerhub##*.}; p++)); do
      [ "${count["${minor}:${p}"]:-0}" -eq "${total[${minor}]}" ] && best="${p}"
    done
    LATEST_PATCH_BY_MINOR["${minor}"]="${best:+${minor}.${best}}"
    LATEST_PATCH_BY_MINOR["${minor}"]="${LATEST_PATCH_BY_MINOR[${minor}]:-${current[${minor}]}}"
    echo "FIPS-checked ${minor}: using ${LATEST_PATCH_BY_MINOR[${minor}]} (Docker Hub latest ${dockerhub})"
  done
}

update_external_images() {
  local i src repo minor patch
  for ((i = 0; i < $(img_count external-images.yaml); i++)); do
    src="$(yq -r ".images[${i}].source" external-images.yaml)"
    [[ "${src}" =~ ^(istio/[^:]+):([0-9]+\.[0-9]+)\.[0-9]+-distroless$ ]] || continue
    repo="${BASH_REMATCH[1]}"; minor="${BASH_REMATCH[2]}"
    patch="${LATEST_PATCH_BY_MINOR[${minor}]:-}"
    [ -n "${patch}" ] && yq -i ".images[${i}].source = \"${repo}:${patch}-distroless\"" external-images.yaml
  done
}

update_fips_images() {
  local i src repo patch rev minor desired found source_tag target_tag
  for ((i = 0; i < $(img_count fips-images.yaml); i++)); do
    src="$(yq -r ".images[${i}].source" fips-images.yaml)"
    [[ "${src}" =~ ^(.+):([0-9]+\.[0-9]+\.[0-9]+)(-([0-9]+))?$ ]] || continue
    repo="${BASH_REMATCH[1]}"; patch="${BASH_REMATCH[2]}"; rev="${BASH_REMATCH[4]:-0}"
    minor="${patch%.*}"
    desired="${LATEST_PATCH_BY_MINOR[${minor}]:-${patch}}"

    # Baseline revision: keep current one only when the patch is unchanged.
    [ "${desired}" = "${patch}" ] || rev=0
    found="$(registry_tags "${repo}" | highest_rev "${desired}")"
    [ -n "${found}" ] && (( found > rev )) && rev="${found}"

    (( rev > 0 )) && source_tag="${desired}-${rev}" || source_tag="${desired}"
    target_tag="${desired}-$((rev + 1))"

    echo "FIPS $(basename "${repo}"): source ${source_tag}, target ${target_tag}"
    yq -i ".images[${i}].source = \"${repo}:${source_tag}\"" fips-images.yaml
    yq -i ".images[${i}].target.tag = \"${target_tag}\"" fips-images.yaml
  done
}

main() {
  ensure_tools
  build_latest_patch_map
  adjust_patch_map_with_fips
  update_external_images
  update_fips_images
  unset REGCERT_JSON
}

main
