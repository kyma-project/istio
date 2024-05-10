# Istio ingress-gateway Deployment, istiod Deployment or Istio CNI DaemonSets are not reconciling back to the desired state after being modified

## Symptoms

- The Istio Ingress Gateway Deployment, Istiod Deployment, or istio-cni-node DaemonSets are not getting up
- Something new is stuck in the pod, and does not get up
- Istio Module Operator return error on Istio Install phase

## Causes

- The Istio Ingress Gateway Deployment, Istiod Deployment, or istio-cni-node DaemonSets pods template is modified, the new container is injected. The new container cannot get up in the pod for some reason, so the pod does not have status Running, and the install phase of the reconciliation fails.

## Remedy

1. Check if a given resource's pod template is modified with additional container
2. Check if the new injected container is not getting up in the pod
3. Remove new additional injected container from the pod template in the given resource
4. Now new pods should work as expected.
5. Istio reconciliation loop should now pass the Installation step

