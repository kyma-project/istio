#!/usr/bin/env bash

# This script is deprecated, because most of the logic is already covered by enable-module.sh script linked in the readme.
set -eo pipefail

# Verify Istio module template is available on cluster
istio_module_template_count=$(kubectl get moduletemplates.operator.kyma-project.io -A --output json | jq '.items | map(. | select(.spec.data.kind=="Istio")) | length')

if [ "$istio_module_template_count" -eq 0 ]; then
  echo "Istio module template is not available on cluster"
  exit 1
fi

# Fetch Kyma CR name managed by lifecycle-manager
kyma_cr_name=$(kubectl get kyma -n kyma-system --no-headers -o custom-columns=":metadata.name")

# Fetch all modules, if modules is not defined, fallback to an empty array and count the number modules that have the name "istio"
istio_module_count=$(kubectl get kyma "$kyma_cr_name" -n kyma-system -o json | jq '.spec.modules | if . == null then [] else . end | map(. | select(.name=="istio")) | length')

if [  "$istio_module_count" -gt 0 ]; then
  echo "Istio module already present on Kyma CR, skipping migration"
  exit 0
fi

kyma_cr_modules=$(kubectl get kyma "$kyma_cr_name" -n kyma-system -o json | jq '.spec.modules')
if [ "$kyma_cr_modules" == "null" ]; then
  echo "No modules defined on Kyma CR yet, initializing modules array"
  kubectl patch kyma "$kyma_cr_name" -n kyma-system --type='json' -p='[{"op": "add", "path": "/spec/modules", "value": [] }]'
fi

echo "Proceeding with migration by adding Istio module to Kyma CR $kyma_cr_name"
kubectl patch kyma "$kyma_cr_name" -n kyma-system --type='json' -p='[{"op": "add", "path": "/spec/modules/-", "value": {"name": "istio"} }]'
# Now we are giving 15 sec of sleep for LM to update Istio Deployment.
sleep 15

number=1
while [[ $number -le 100 ]]; do
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
