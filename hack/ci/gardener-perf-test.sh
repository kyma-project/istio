#!/usr/bin/env bash

#
##Description: This script provisions a Gardener cluster with config specified in environmental variables and runs Istio module integration tests

set -euo pipefail

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
    GARDENER_PROVIDER
    GARDENER_REGION
    GARDENER_ZONES
    GARDENER_KYMA_PROW_KUBECONFIG
    GARDENER_KYMA_PROW_PROJECT_NAME
    GARDENER_KYMA_PROW_PROVIDER_SECRET_NAME
    GARDENER_CLUSTER_VERSION
    MACHINE_TYPE
    DISK_SIZE
    DISK_TYPE
    SCALER_MAX
    SCALER_MIN
)

check_required_vars "${requiredVars[@]}"

function cleanup() {
  kubectl annotate shoot "${CLUSTER_NAME}" confirmation.gardener.cloud/deletion=true \
      --overwrite \
      -n "garden-${GARDENER_KYMA_PROW_PROJECT_NAME}" \
      --kubeconfig "${GARDENER_KYMA_PROW_KUBECONFIG}"

  kubectl delete shoot "${CLUSTER_NAME}" \
    --wait="false" \
    --kubeconfig "${GARDENER_KYMA_PROW_KUBECONFIG}" \
    -n "garden-${GARDENER_KYMA_PROW_PROJECT_NAME}"
}

# Cleanup on exit, be it successful or on fail
trap cleanup EXIT INT

# Install Kyma CLI in latest version
echo "--> Install kyma CLI locally to /tmp/bin"
curl -Lo kyma.tar.gz "https://github.com/kyma-project/cli/releases/latest/download/kyma_linux_x86_64.tar.gz" \
&& tar -zxvf kyma.tar.gz && chmod +x kyma \
&& rm -f kyma.tar.gz
chmod +x kyma

# Add pwd to path to be able to use Kyma binary
export PATH="${PATH}:${PWD}"

# Provision gardener cluster
CLUSTER_NAME=ag-$(echo $RANDOM | md5sum | head -c 7)

kyma version --client
kyma provision gardener ${GARDENER_PROVIDER} \
        --secret "${GARDENER_KYMA_PROW_PROVIDER_SECRET_NAME}" \
        --name "${CLUSTER_NAME}" \
        --project "${GARDENER_KYMA_PROW_PROJECT_NAME}" \
        --credentials "${GARDENER_KYMA_PROW_KUBECONFIG}" \
        --region "${GARDENER_REGION}" \
        --zones "${GARDENER_ZONES}" \
        --type "${MACHINE_TYPE}" \
        --disk-size $DISK_SIZE \
        --disk-type "${DISK_TYPE}" \
        --scaler-max $SCALER_MAX \
        --scaler-min $SCALER_MIN \
        --kube-version="${GARDENER_CLUSTER_VERSION}" \
        --attempts 3 \
        --verbose

tag=$(gcloud container images list-tags europe-docker.pkg.dev/kyma-project/prod/istio-manager --limit 1 --format json | jq '.[0].tags[1]')
IMG=europe-docker.pkg.dev/kyma-project/prod/istio-manager:${tag} make install deploy
kubectl apply -f config/samples/operator_v1alpha1_istio.yaml

number=1
	while [[ $number -le 100 ]] ; do
		echo ">--> checking kyma status #$number"
		STATUS=$(kubectl get istio istio-sample -o jsonpath='{.status.state}')
		echo "kyma status: ${STATUS:='UNKNOWN'}"
		[[ "$STATUS" == "Ready" ]] && return 0
		sleep 5
        	((number = number + 1))
	done

	kubectl get all --all-namespaces
exit 1

cd performance_tests && make test-performance