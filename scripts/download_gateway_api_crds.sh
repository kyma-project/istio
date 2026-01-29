#!/bin/bash

# Script to download Gateway API CRDs
# This should be run to update the embedded Gateway API CRDs

set -e

VERSION="v1.4.1"
OUTPUT_FILE="internal/reconciliations/istio/gateway-api-crds.yaml"

echo "Downloading Gateway API CRDs version ${VERSION}..."

curl -L "https://github.com/kubernetes-sigs/gateway-api/releases/download/${VERSION}/standard-install.yaml" -o "${OUTPUT_FILE}"

echo "Gateway API CRDs downloaded to ${OUTPUT_FILE}"
echo "Version: ${VERSION}"
