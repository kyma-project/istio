#!/usr/bin/env bash

set -eo pipefail

# Fetch Kyma CR name managed by lifecycle-manager
kyma_cr_name=$(kubectl get kyma -n kyma-system --no-headers -o custom-columns=":metadata.name")

# Check if Istio is already present on Kyma CR
istio_module_count=$(kubectl get kyma "$kyma_cr_name" -n kyma-system -o json | jq '.spec.modules | if . == null then [] else . end | map(. | select(.name=="istio")) | length')

# Check if there is Istio in KymaCR set to remoteModuleRefTemplate, since we don't want to affect any other clusters containing Istio
kyma_contains_local_moduletemplate_config=$(kubectl get -n kyma-system kyma default -o json | jq '.spec.modules[] | select(.remoteModuleTemplateRef == "kyma-system/istio-migration-test-fast") | any')

# Reverting KymaCR
if [  "$istio_module_count" -eq 0 ]; then
  echo "Istio module is not present on Kyma CR, nothing to revert. Skipping to ModuleTemplate deletion"
elif [ "$kyma_contains_local_moduletemplate_config" != true ]; then
  echo "Istio module is not set to remoteModuleTemplateRef in the given cluster"
else
  echo "Setting Istio Module to the default channel."
  INDEX=$(kubectl get -n kyma-system kyma default -o json  | jq '.spec.modules | map(.name == "istio") | index(true)')
  kubectl patch kyma "$kyma_cr_name" -n kyma-system --type='json' -p="[{'op': 'replace', 'path': '/spec/modules/$INDEX', 'value': {'name': 'istio'} }]"
fi

# Deleting testing purpose local ModuleTemplate
kubectl delete -n kyma-system moduletemplate istio-migration-test-fast || echo "Testing purpose ModuleTemplate not found"
