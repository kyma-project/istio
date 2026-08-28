#!/usr/bin/env bash
#
# Install the prerequisites the tests/e2e/tests/* suites need to
# exercise dualstack on a Gardener AWS shoot with
# `dualStack.enabled=true` and `ipFamilies=[IPv4, IPv6]`:
#   1. Create kyma-system namespace to be able to create ConfigMap.
#   2. Apply the `kyma-provisioning-info` ConfigMap that
#      istio-controller-manager reads to gate dualstack
#      (internal/clusterconfig/clusterconfig.go IsDualStackEnabled).
#      On real BTP clusters this is written by infrastructure-manager;
#      Gardener-native CI shoots (and local) have to fill it
#      in themselves.
#
# Idempotent. Safe to re-run.
#
# Uses whatever KUBECONFIG the caller has set.

kubectl create namespace kyma-system --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace kyma-system istio-injection=enabled --overwrite

printf 'networkDetails:\n  dualStackIPEnabled: true\n' \
  | kubectl create configmap -n kyma-system kyma-provisioning-info \
      --from-file=details=/dev/stdin \
      --dry-run=client -o yaml \
  | kubectl apply -f -