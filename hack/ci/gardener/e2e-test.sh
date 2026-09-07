#!/usr/bin/env bash

# Description: This script runs given integration tests on a real Gardener cluster
# It installs istio and api gateway and then runs make test targets provided via commandline arguments to that script
# It requires the following env variables:
# - IMG - API gateway image to be deployed (by make deploy)
# - CLUSTER_NAME - Gardener cluster name
# - CLUSTER_KUBECONFIG - Gardener cluster kubeconfig path
# - GARDENER_CONFIGURATION - configuration preset; provides GARDENER_IP_STACK
#   via configurations/${GARDENER_CONFIGURATION}/vars.sh. When the shoot is
#   dualstack (GARDENER_IP_STACK=dualstack) the experimental istio-manager is
#   installed (the only build that honours dualStackIPEnabled).
# Optional:
# - TEST_IP_FAMILY - ipv4 (default) | ipv6 | dualstack. Selects only the test
#   client dial family; it does NOT drive infra dualstack (that is the shoot's
#   GARDENER_IP_STACK, above).

set -eo pipefail
script_dir="$(dirname "$(readlink -f "$0")")"
# shellcheck source=../common.sh
source "${script_dir}/../common.sh"

require_positional make_target "$1"

require_vars IMG CLUSTER_NAME CLUSTER_KUBECONFIG GARDENER_CONFIGURATION

load_configuration "${GARDENER_CONFIGURATION}"

require_vars GARDENER_IP_STACK

echo "Make target: ${make_target}"

echo "Executing tests in cluster ${CLUSTER_NAME}, kubeconfig ${CLUSTER_KUBECONFIG}"
export KUBECONFIG="${CLUSTER_KUBECONFIG}"

export CLUSTER_DOMAIN=$(kubectl get configmap -n kube-system shoot-info -o jsonpath="{.data.domain}")
echo "Cluster domain: ${CLUSTER_DOMAIN}"

export GARDENER_PROVIDER=$(kubectl get configmap -n kube-system shoot-info -o jsonpath="{.data.provider}")
echo "Gardener provider: ${GARDENER_PROVIDER}"

export IS_GARDENER=true # this variable is used in tests to make decisions based on the fact that the tests are running in Gardener

[[ "${GARDENER_IP_STACK}" == "dualstack" ]] && DUAL_STACK_ENABLED="true" || DUAL_STACK_ENABLED="false"
echo "Dual stack enabled: ${DUAL_STACK_ENABLED}"

# Add pwd to path to be able to use binaries downloaded in scripts
export PATH="${PATH}:${PWD}"

start_group "Creating kyma-system namespace"
make create-namespace
end_group

start_group "Creating kyma-provisioning-info configmap"
make create-provisioning-info DUAL_STACK_ENABLED="${DUAL_STACK_ENABLED}"
end_group

start_group "Deploying module, image: ${IMG}"
make deploy
end_group

start_group "Executing tests: ${make_target}"
make "${make_target}"
end_group
