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

./hack/ci/provision-gardener.sh
# Cleanup on exit, be it successful or on fail
trap cleanup EXIT INT

tag=$(gcloud container images list-tags europe-docker.pkg.dev/kyma-project/prod/istio-manager --limit 1 --format json | jq '.[0].tags[1]')
IMG=europe-docker.pkg.dev/kyma-project/prod/istio-manager:${tag} make install deploy

cd tests/performance || exit
make test-performance
