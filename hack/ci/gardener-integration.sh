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

export MAKE_TEST_TARGET="${MAKE_TEST_TARGET:-istio-integration-test}"
make install deploy "$MAKE_TEST_TARGET"
