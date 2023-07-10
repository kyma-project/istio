#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

RELEASE_INFO=$(curl -L \
                 -H "Accept: application/vnd.github+json" \
                 -H "X-GitHub-Api-Version: 2022-11-28" \
                 https://api.github.com/repos/kyma-project/istio/releases/latest)

RELEASE_MANIFEST_URL=$(echo "$RELEASE_INFO" | jq -r '.assets[] | select(.name == "istio-manager.yaml") | .browser_download_url')
curl -L "$RELEASE_MANIFEST_URL" | kubectl apply -f -