# Changes to Istio Ingress Gateway Deployment, Istiod Deployment, or Istio CNI DaemonSets Are Not Reverted After Reconciliation

## Symptoms

- The Istio Ingress Gateway Deployment, Istiod Deployment, or  Istio CNI DaemonSets fails to start.
- Any of these resources contains a non-default container that is unable to start.
- The Istio custom resource is in the `Error` state.

## Cause

- The Istio Ingress Gateway Deployment, Istiod Deployment, or istio-cni-node DaemonSets might have a container injected by an outside component, for example with a mutating webhook. If the new container cannot start for any reason, the pod will not get to a Running state, failing Istio module reconciliation.

## Remedy

1. Check if a given resource's Pod template has been modified to include an additional container.
2. Check if the new injected container is not getting up in the pod
3. If the container is unable to start, remove it from the Pod template of the given resource.
4. If the error persists, check if there is a mutating webhook in the cluster that is modifying Istio resources.

