#!/usr/bin/env bash

CLUSTER_NAME=gi-$(echo $RANDOM | md5sum | head -c 7)
export CLUSTER_NAME

function cleanup() {
    kubectl annotate shoot "${CLUSTER_NAME}" confirmation.gardener.cloud/deletion=true \
          --overwrite \
          -n "garden-${GARDENER_PROJECT_NAME}" \
          --kubeconfig "${GARDENER_KUBECONFIG}"

    kubectl delete shoot "${CLUSTER_NAME}" \
      --wait="false" \
      --kubeconfig "${GARDENER_KUBECONFIG}" \
      -n "garden-${GARDENER_PROJECT_NAME}"

    exit
}

./hack/ci/provision-gardener.sh
# Cleanup on exit, be it successful or on fail
trap cleanup EXIT INT

./hack/ci/jobguard.sh

if [ "$JOB_TYPE" == "presubmit" ]; then
  export IMG=europe-docker.pkg.dev/kyma-project/dev/istio-manager:PR-${PULL_NUMBER}
elif [ "$JOB_TYPE" == "postsubmit" ]; then
  POST_IMAGE_VERSION=v$(shell date '+%Y%m%d')-$(shell printf %.8s ${PULL_BASE_SHA})
  export IMG=europe-docker.pkg.dev/kyma-project/prod/istio-manager:${POST_IMAGE_VERSION}
fi

make install deploy istio-integration-test
