#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

TARGET_BRANCH="$1"

# The if statement is here to make the script work properly on release branches
# Example: if we want to patch 0.2.0 to 0.2.1 when already 0.3.0 exists it makes
# the script deploy 0.2.0 instead 0.3.0 so we don't test upgrade from 0.3.0 to 0.2.1
if [ "$TARGET_BRANCH" != "main" ] && [ "$TARGET_BRANCH" != "" ]
then
  TAG=$(git describe --tags --abbrev=0)
  RELEASE_MANIFEST_URL="https://github.com/kyma-project/istio/releases/download/${TAG}/istio-manager.yaml"
  curl -L "$RELEASE_MANIFEST_URL" | kubectl apply -f -
else
 RELEASE_INFO=$(curl -L \
                  -H "Accept: application/vnd.github+json" \
                  -H "X-GitHub-Api-Version: 2022-11-28" \
                  https://api.github.com/repos/kyma-project/istio/releases/latest)

 RELEASE_MANIFEST_URL=$(echo "$RELEASE_INFO" | jq -r '.assets[] | select(.name == "istio-manager.yaml") | .browser_download_url')
 curl -L "$RELEASE_MANIFEST_URL" | kubectl apply -f -
fi