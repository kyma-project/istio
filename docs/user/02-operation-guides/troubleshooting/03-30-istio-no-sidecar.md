# Issues with Istio sidecar injection

## Symptom

A Pod doesn't have a sidecar but you did not disable sidecar injection on purpose.

## Cause

By default, Kyma Istio Operator has sidecar injection disabled - there is no automatic sidecar injection into any Pod in a cluster.

## Remedy

Use this [guide](../operations/02-10-check-if-sidecar-injection-is-enabled.md) to check if your Pod and its Namespace have Istio sidecar injected. Learn how to [enable Istio sidecar proxy injection](../operations/02-20-enable-sidecar-injection.md).