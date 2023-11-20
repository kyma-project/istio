#!/usr/bin/env bash

set -eo pipefail

# Since we are using relative paths in this script we need to change directory to the script's to not depend on current working directory during the execution
script_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$script_path"

# Fetch Kyma CR name managed by lifecycle-manager
kyma_cr_name=$(kubectl get kyma -n kyma-system --no-headers -o custom-columns=":metadata.name")

# Check if Istio is already present on Kyma CR
istio_module_count=$(kubectl get kyma "$kyma_cr_name" -n kyma-system -o json | jq '.spec.modules | if . == null then [] else . end | map(. | select(.name=="istio")) | length')

if [  "$istio_module_count" -gt 0 ]; then
  echo "Istio module already present on Kyma CR, skipping migration"
  exit 0
fi

# Fetch all modules, if modules is not defined, fallback to an empty array and count the number modules that have the name "istio"

kyma_cr_modules=$(kubectl get kyma "$kyma_cr_name" -n kyma-system -o json | jq '.spec.modules')
if [ "$kyma_cr_modules" == "null" ]; then
  echo "No modules defined on Kyma CR yet, initializing modules array"
  kubectl patch kyma "$kyma_cr_name" -n kyma-system --type='json' -p='[{"op": "add", "path": "/spec/modules", "value": [] }]'
fi

(kubectl get -n kyma-system moduletemplate istio-migration-test-fast 2>/dev/null && (echo "Testing module template already on the cluster" ; exit 1)) || echo "Testing ModuleTemplate not found yet. Proceeding to apply one."

echo "Applying local ModuleTemplate with fast channel to the cluster"
echo "$script_path"
kubectl apply -f ./module-template-migration-test-fast.yaml

echo "Proceeding with migration by adding Istio module to Kyma CR $kyma_cr_name"
kubectl patch kyma "$kyma_cr_name" -n kyma-system --type='json' -p='[{"op": "add", "path": "/spec/modules/-", "value": {"name": "istio", "remoteModuleTemplateRef": "kyma-system/istio-migration-test-fast"} }]'

number=1
while [[ $number -le 100 ]]; do
  replicas=$(kubectl get -n kyma-system deployment istio-controller-manager -o json | jq '.status.replicas')
  readyReplicas=$(kubectl get -n kyma-system deployment istio-controller-manager -o json | jq '.status.readyReplicas')
  if [ "$readyReplicas" -ne "$replicas" ]; then
    ((number = number + 1))
    echo "Istio Deployment not ready yet."
    continue
  fi
  STATUS=$(kubectl -n kyma-system get istio default -o jsonpath='{.status.state}' || echo " failed retrieving default Istio CR")
  ISTIO_CR_COUNT=$(kubectl get istios -n kyma-system --output json | jq '.items | length')

  if [ "$STATUS" == "Ready" ]; then
    echo "Migration successful"
    exit 0
  elif [ "$STATUS" == "Error" ] && [ "$ISTIO_CR_COUNT" -gt 1 ]; then
    echo "More than one Istio CR present on the cluster. Script rename-to-default.sh might be required."
    exit 1
  fi

  sleep 5
  ((number = number + 1))
done

echo "Migration failed"
exit 2
