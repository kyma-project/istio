#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

TARGET_BRANCH="$1"

# If the first tag in branch history doesn't match release tag minor (or target is main)
# install istio from the latest release instead
TAG=$(git describe --tags --abbrev=0)
if [ "${TAG%.*}" == "${TARGET_BRANCH#release\-}" ]
then
  TAG="${TAG%-experimental}"
  echo "Installing Istio ${TAG}"
  RELEASE_MANIFEST_URL="https://github.com/kyma-project/istio/releases/download/${TAG}/istio-manager.yaml"
  curl -L "$RELEASE_MANIFEST_URL" | kubectl apply -f -
else
  echo "Installing Istio from latest release"
  RELEASE_MANIFEST_URL="https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml"
  curl -L "$RELEASE_MANIFEST_URL" | kubectl apply -f -
fi
