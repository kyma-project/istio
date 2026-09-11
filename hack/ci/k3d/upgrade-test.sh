#!/usr/bin/env bash

# Description: This script runs upgrade tests on a k3d cluster.
# It deploys a reference version of the operator and then runs the given make target
# It requires the following env variables:
# - REFERENCE_RELEASE - GitHub release to deploy as the baseline before the upgrade
# - IMG - An operator image to be deployed by the upgrade test
# - K3D_CONFIGURATION - configuration preset, used to load configurations/${K3D_CONFIGURATION}/vars.sh

set -eo pipefail
script_dir="$(dirname "$(readlink -f "$0")")"
# shellcheck source=../common.sh
source "${script_dir}/../common.sh"

require_positional make_target "$1"

require_vars K3D_CONFIGURATION REFERENCE_RELEASE IMG

load_configuration "${K3D_CONFIGURATION}"

export IS_GARDENER=false

# Add pwd to path to be able to use binaries downloaded in scripts
export PATH="${PATH}:${PWD}"

start_group "Creating kyma-system namespace"
make create-namespace
end_group

start_group "Creating kyma-provisioning-info configmap"
make create-provisioning-info DUAL_STACK_ENABLED=false
end_group

start_group "Deploy reference release of a module: ${REFERENCE_RELEASE}"
make deploy-release RELEASE_VERSION="${REFERENCE_RELEASE}"
end_group

start_group "Executing tests: ${make_target}"
make "${make_target}"
end_group
