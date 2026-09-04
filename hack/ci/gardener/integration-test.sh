#!/usr/bin/env bash

# Description: This script runs given integration tests on a real Gardener cluster
# It deploys Istio module and then runs make test targets provided via commandline arguments to that script
# It requires the following env variables:
# - IMG - Istio module image to be deployed (by make deploy)
# - CLUSTER_NAME - Gardener cluster name
# - CLUSTER_KUBECONFIG - Gardener cluster kubeconfig path
# - GARDENER_CONFIGURATION - provisioning preset; provides GARDENER_PROVIDER / GARDENER_IP_STACK
#   via configurations/${GARDENER_CONFIGURATION}/vars.sh
# - TEST_IP_FAMILY - Tests can be run to check (ipv4|ipv6|dualstack)

set -eo pipefail

script_dir="$(dirname "$(readlink -f "$0")")"

if [ $# -lt 1 ]; then
    >&2 echo "Make target is required as parameter"
    exit 1
fi

function check_required_vars() {
  local requiredVarMissing=false
  for var in "$@"; do
    if [ -z "${!var}" ]; then
      >&2 echo "Environment variable ${var} is required but not set"
      requiredVarMissing=true
    fi
  done
  if [ "${requiredVarMissing}" = true ] ; then
    exit 2
  fi
}

requiredVars=(
    IMG
    CLUSTER_NAME
    CLUSTER_KUBECONFIG
    TEST_IP_FAMILY
    GARDENER_CONFIGURATION
)

check_required_vars "${requiredVars[@]}"

# Load provider / IP stack from the same preset used to provision the cluster,
# so provisioning and test-time selection share a single source of truth.
preset_vars="${script_dir}/configurations/${GARDENER_CONFIGURATION}/vars.sh"
if [ ! -f "${preset_vars}" ]; then
    >&2 echo "File '${preset_vars}' required but not found"
    exit 2
fi
set -a
source "${preset_vars}"
set +a

make_target="$1"

if [ -z "$make_target" ]; then
    echo "Make target is required as parameter"
    exit 3
fi

echo "Make target: $make_target"

echo "Executing tests in cluster ${CLUSTER_NAME}, kubeconfig ${CLUSTER_KUBECONFIG}"
export KUBECONFIG="${CLUSTER_KUBECONFIG}"

export CLUSTER_DOMAIN=$(kubectl get configmap -n kube-system shoot-info -o jsonpath="{.data.domain}")
echo "Cluster domain: ${CLUSTER_DOMAIN}"

export GARDENER_PROVIDER=$(kubectl get configmap -n kube-system shoot-info -o jsonpath="{.data.provider}")
echo "Gardener provider: ${GARDENER_PROVIDER}"

export TEST_DOMAIN="${CLUSTER_DOMAIN}"

echo "Creating kyma-system namespace and kyma-provisioning-info configmap "

[[ "${GARDENER_IP_STACK}" == "dualstack" ]] && DUAL_STACK_ENABLED="true" || DUAL_STACK_ENABLED="false"

make create-provisioning-info DUAL_STACK_ENABLED="${DUAL_STACK_ENABLED}"

# Add pwd to path to be able to use binaries downloaded in scripts
export PATH="${PATH}:${PWD}"

echo "Deploying Istio module, image: ${IMG}"
make deploy


echo "Executing tests..."
echo "Executing make target $make_target"
make "$make_target"
echo "Tests finished"
