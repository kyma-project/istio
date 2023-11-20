#!/usr/bin/env bash

set -eo pipefail

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
