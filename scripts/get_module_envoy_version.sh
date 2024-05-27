#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

if [ "$#" -lt 1 ]; then
    echo "The Istio tag must be provided as the first argument"
    exit 1
fi

ISTIO_TAG=$1

# The command will have a return in the following format: /usr/local/bin/envoy  version: bcf6c19288e9d4a133f815657c951539018bc9bb/1.29.4-dev/Clean/RELEASE/BoringSSL
# We need to sanitize the version by removing newlines
docker run --rm --entrypoint "/usr/local/bin/envoy" europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2:${ISTIO_TAG}-distroless --version | awk '{print $3}' | awk -F'/' '{print $2}' | tr -d '\n'

