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
kubectl apply -f config/samples/operator_v1alpha2_istio.yaml

number=1
	while [[ $number -le 100 ]] ; do
		echo ">--> checking kyma status #$number"
		STATUS=$(kubectl get -n kyma-system istio default -o jsonpath='{.status.state}')
		echo "kyma status: ${STATUS:='UNKNOWN'}"
		[[ "$STATUS" == "Ready" ]] && break
		sleep 5
        	((number = number + 1))
	done

domain=$(kubectl config view -o json | jq '.clusters[0].cluster.server' | sed -e "s/https:\/\/api.//" -e 's/"//g')
kubectl annotate service -n istio-system istio-ingressgateway "dns.gardener.cloud/dnsnames=*.${domain}" --overwrite

cd tests/performance || exit

n=0
until [ "$n" -ge 5 ]
do
   make test-performance && break
   n=$((n+1))
   sleep 15
done

