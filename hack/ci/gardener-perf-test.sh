#!/usr/bin/env bash

CLUSTER_NAME=gp-$(echo $RANDOM | md5sum | head -c 7)
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

# Cleanup on exit, be it successful or on fail
trap cleanup EXIT INT

./hack/ci/provision-gardener.sh

echo "waiting for Gardener to finish shoot reconcile..."
kubectl wait --kubeconfig "${GARDENER_KUBECONFIG}" --for=jsonpath='{.status.lastOperation.state}'=Succeeded --timeout=600s "shoots/${CLUSTER_NAME}"

make install deploy

cd tests/performance || exit
make test-performance-web
