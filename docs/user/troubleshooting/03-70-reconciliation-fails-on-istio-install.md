# Istio ingress-gateway Deployment, istiod Deployment or Istio CNI DaemonSets are not reconciling back to the desired state after being modified

## Symptoms

- The Istio Ingress Gateway Deployment, Istiod Deployment, or istio-cni-node DaemonSets are not getting up
- A non-default container is present in any of the above, and cannot start
- Istio Custom Resource is in Error state

## Causes

- The Istio Ingress Gateway Deployment, Istiod Deployment, or istio-cni-node DaemonSets might have a container injected by an outside component, for example with a mutating webhook. If the new container cannot start for any reason, the pod will not get to a Running state, failing Istio module reconciliation.

## Remedy

1. Check if a given resource's pod template is modified with additional container
2. Check if the new injected container is not getting up in the pod
3. Remove new additional injected container from the pod template in the given resource
4. If error persists, check if there might be a mutating webhook in the cluster that is modifying Istio resources

