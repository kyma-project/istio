#!/usr/bin/env bash

# Prints the Istio version bundled in this module to stdout.
# Reads it from internal/istiooperator/istio-operator.yaml.

set -eo pipefail
script_dir="$(dirname "$(readlink -f "$0")")"

grep "tag:" "${script_dir}/../internal/istiooperator/istio-operator.yaml" \
  | awk '{print $2}' | tr -d '"' | awk -F'-' '{print $1}'
