# Troubleshooting: The Istio module was unintentionally disabled
Follow the steps outlined in this troubleshooting guide if you unintentionally deleted the Istio module and want to restore the system to its normal state without losing any user-created resources. However, if you intended to delete the module, the symptoms described in this document are expected and you must clean up the orphaned resources by yourself.

## Symptom

* The Istio custom resource (CR) is in the `WARNING` state.


### Typical log output / error messages

```
kubectl get istio -n kyma-system
NAME      STATE
default   Warning
```
```
kubectl get istio default -n kyma-system -o jsonpath='{.status.description}'
Resources blocking deletion: DestinationRule:kyma-system/api-gateway-metrics;DestinationRule:kyma-system/eventing-nats;PeerAuthentication:kyma-system/eventing-controller-metrics;PeerAuthentication:kyma-system/eventing-publisher-proxy-metrics
```

## Cause

- The Istio module was disabled, but it was not completely removed because the user's CRs still exist.

For example, the issue occurs when you delete the Istio module, but there are still `VirtualService` resources that either belong to the user or were installed by another Kyma component or module. In such cases, the hooked finalizer pauses the deletion of the module until you remove all the related resources. This [blocking deletion strategy](https://github.com/kyma-project/community/issues/765) is intentionally designed and is enabled by default for the Istio module.


## Remedy

 1. Edit the Istio CR and remove the finalizer.
```
kubectl edit istio -n kyma-system default
```
```diff
apiVersion: operator.kyma-project.io/v1alpha1
kind: Istio
metadata:
< finalizers:
< - istios.operator.kyma-project.io/istio-installation
  generation: 2
  name: default
  namespace: kyma-system

spec:
# ...
status:
  description: 'Resources blocking deletion: DestinationRule:kyma-system/api-gateway-metrics;DestinationRule:kyma-system/eventing-nats;PeerAuthentication:kyma-system/eventing-controller-metrics;PeerAuthentication:kyma-system/eventing-publisher-proxy-metrics'
  state: Warning
```
 2. When the finalizer is removed, the Istio CR is deleted. Other resources, such as the `istiod` deployment, remain on the cluster.

 3. Reapply the Istio CR to enable the Istio module once again.

By completing the steps, the module's reconciliation is triggered again. The Istio CR should return to the `READY` state within a few seconds.
