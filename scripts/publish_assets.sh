#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked
set -x

RELEASE_TAG=$1
RELEASE_ID=$2

REPOSITORY=${REPOSITORY:-kyma-project/istio}
GITHUB_URL=https://uploads.github.com/repos/${REPOSITORY}
GITHUB_AUTH_HEADER="Authorization: Bearer ${GITHUB_TOKEN}"
IMG="europe-docker.pkg.dev/kyma-project/prod/istio/releases/istio-manager:${RELEASE_TAG}"
VERSION="${RELEASE_TAG}"

IMG="${IMG}" VERSION="${VERSION}" make generate-manifests
curl -f -L \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "${GITHUB_AUTH_HEADER}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @"istio-manager.yaml" \
  ${GITHUB_URL}/releases/${RELEASE_ID}/assets?name=istio-manager.yaml

IMG="${IMG}-experimental" VERSION="${VERSION}-experimental" make generate-manifests
curl -f -L \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "${GITHUB_AUTH_HEADER}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @"istio-manager.yaml" \
  ${GITHUB_URL}/releases/${RELEASE_ID}/assets?name=istio-manager-experimental.yaml

curl -f -L \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "${GITHUB_AUTH_HEADER}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @"config/samples/operator_v1alpha2_istio.yaml" \
  ${GITHUB_URL}/releases/${RELEASE_ID}/assets?name=istio-default-cr.yaml
