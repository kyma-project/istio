#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Check the istio-operator.yaml for the string "tag:" and extract the string after it
# The command will have a return in the following format: tag: 1.11.0

# - Find the line containing the "tag:" in the the istio-operator.yaml
# - Extract the tag
# - Remove " from the string
# - Remove the -distroless in the tag
cat internal/istiooperator/istio-operator.yaml | grep "tag:" | awk '{print $2}' | tr -d '"' | awk -F'-' '{print $1}'

