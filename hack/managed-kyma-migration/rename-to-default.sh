#!/usr/bin/env bash

# Script needs to be used when Istio CustomResource is in Ready state. If Istio is currently in Processing, script will fail fast, as we cannot determine which Istio is being currently reconciled

set -eo pipefail

kubectl scale -n kyma-system deployment/istio-controller-manager --replicas 0

istio_ready=$(kubectl get istio -n kyma-system --output json | jq '.items[] | select((.status.state=="Ready") or (.status.state=="Warning"))')

if [ "$istio_ready" == "" ]; then
  echo "No 'Ready' istio found. Make sure that your Istio CustomResource is not in 'Processing'"
  exit 0
fi

ready_istio_name=$(echo "$istio_ready" | jq '.metadata.name')

if [ "$ready_istio_name" == "default" ]; then
  echo "no changes required"
else
  updated_istio=$(echo "$istio_ready" | jq '.metadata.name |= "default" | del(.metadata.annotations) | del(.metadata.creationTimestamp) | del(.metadata.generation) |del(.metadata.finalizers) | del(.metadata.uid) | del(.metadata.resourceVersion)')
  echo $updated_istio | kubectl apply -f -
fi

kubectl get istio -n kyma-system --field-selector 'metadata.name!=default' -o=json | jq '.items[].metadata.finalizers = null' | kubectl apply -f -
kubectl delete istio -n kyma-system --field-selector 'metadata.name!=default'

kubectl scale -n kyma-system deployment/istio-controller-manager --replicas 1
