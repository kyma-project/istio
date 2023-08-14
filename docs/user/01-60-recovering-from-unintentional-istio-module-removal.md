# Troubleshooting: Module was disabled unintentionally

## Symptom

* Istio CR is in `WARNING` state]


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

- Module was disabled. It was not effectively removed because user's custom resources still exist. 

For instance: istio module was deleted but there are still `VirtualService` resources belonging to the user or installed by another Kyma component/module. In such case the hooked finalizer pauses normal module deletion until user cleans up all the related resources. This [blocking deletion strategy](https://github.com/kyma-project/community/issues/765) is designed on purpose and is enabled by default for all modules.


## Remedy

> NOTE: The following assumes that (for any reason) the module deletion was unintened and the goal  is to recover normal system state w/o loosing any user created resources. In case module deletion WAS intended, the symptoms described above are expected and the user should clean up the orphaned resources by himself. 


 1. Edit the Module CR (in this case, Keda CR) and remove the finalizer.
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
 2. The the Module CR should be deleted, but no resources will be removed (for example istiod deployment will stay on the cluster)

 3. Enable the module back by reapplying the custom resource

This will retrigger reconcilation of the module. Istio CR should be back in `READY` state in a few seconds.
