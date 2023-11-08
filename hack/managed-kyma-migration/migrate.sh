#!/usr/bin/env bash

set -eo pipefail

function check_apigateway_status () {
	local number=1
	while [[ $number -le 100 ]] ; do
		echo ">--> checking kyma status #$number"
		local STATUS=$(kubectl get apigateway default -o jsonpath='{.status.state}')
		echo "apigateway status: ${STATUS:='UNKNOWN'}"
		[[ "$STATUS" == "Ready" ]] && return 0
		sleep 5
        	((number = number + 1))
	done

	kubectl get all --all-namespaces
	exit 1
}

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

# Check if Istio CR is already present on Kubernetes cluster
istio_crs_count=$(kubectl get istios -n kyma-system --output json | jq '.items | length')

if [ "$istio_crs_count" -gt 1 ]; then
  echo "WARNING: Multiple Istio CRs found"
fi

kyma_cr_modules=$(kubectl get kyma "$kyma_cr_name" -n kyma-system -o json | jq '.spec.modules')
if [ "$kyma_cr_modules" == "null" ]; then
  echo "No modules defined on Kyma CR yet, initializing modules array"
  kubectl patch kyma "$kyma_cr_name" -n kyma-system --type='json' -p='[{"op": "add", "path": "/spec/modules", "value": [] }]'
fi

echo "Proceeding with migration by adding Istio module to Kyma CR $kyma_cr_name"
kubectl patch kyma "$kyma_cr_name" -n kyma-system --type='json' -p='[{"op": "add", "path": "/spec/modules/-", "value": {"name": "istio"} }]'

check_apigateway_status

echo "Istio CR migration completed successfully"
exit 0
