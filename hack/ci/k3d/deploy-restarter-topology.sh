#!/usr/bin/env bash

# Description: Deploys the restarter E2E topology onto an already-provisioned k3d
# cluster. The base istio controller is deployed with --enable-restart=false
# (config/e2e-restarter overlay) and the standalone restarter controller with the
# proxy sidecar resource flags under test (restarter/config/e2e overlay).
#
# Required env variables:
# - BASE_IMG      - base istio-manager image (loaded/imported into the cluster)
# - RESTARTER_IMG - restarter image (loaded/imported into the cluster)

set -eo pipefail
script_dir="$(dirname "$(readlink -f "$0")")"
repo_root="$(cd "${script_dir}/../../.." && pwd)"
# shellcheck source=../common.sh
source "${script_dir}/../common.sh"

require_vars BASE_IMG RESTARTER_IMG

export PATH="${PATH}:${repo_root}"

cd "${repo_root}"

start_group "Creating kyma-system namespace"
make create-namespace
end_group

start_group "Creating kyma-provisioning-info configmap"
make create-provisioning-info DUAL_STACK_ENABLED=false
end_group

# Regenerate config/default/kustomization.yaml, which the e2e overlay references.
start_group "Generating base kustomization"
make module-version
end_group

start_group "Deploying base controller (--enable-restart=false), image: ${BASE_IMG}"
(cd config/manager && kustomize edit set image "controller=${BASE_IMG}")
kustomize build config/e2e-restarter | kubectl apply -f -
end_group

start_group "Deploying restarter controller, image: ${RESTARTER_IMG}"
(cd restarter/config/manager && kustomize edit set image "controller=${RESTARTER_IMG}")
kustomize build --load-restrictor LoadRestrictionsNone restarter/config/e2e | kubectl apply -f -
end_group

start_group "Waiting for controllers to become available"
kubectl rollout status -n kyma-system deployment/istio-controller-manager --timeout=300s
kubectl rollout status -n restarter-system deployment/restarter-controller-manager --timeout=300s
end_group
