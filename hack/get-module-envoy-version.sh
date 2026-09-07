#!/usr/bin/env bash

# Prints the Envoy version bundled in the given Istio release image to stdout.

set -eo pipefail

if [ -z "$1" ]; then
  echo "Usage: $0 <istio-tag>" >&2
  exit 1
fi

istio_tag="$1"
DOCKER_BIN="${DOCKER_BIN:-docker}"

# Output format: /usr/local/bin/envoy  version: <hash>/<version>-dev/Clean/RELEASE/BoringSSL
# Extract the version field and strip the -dev suffix.
${DOCKER_BIN} run --rm --entrypoint "/usr/local/bin/envoy" \
  "europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2:${istio_tag}-distroless" \
  --version \
  | awk '{print $3}' | awk -F'/' '{print $2}' | tr -d '\n' | sed 's/-dev//'
