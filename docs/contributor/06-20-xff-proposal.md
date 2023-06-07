# Istio v1alpha1 proposal

## Summary 

This document proposes Istio Manager API exposed to users. The initial version of the API includes hpa and resources' configuration options for Istio components, such as istiod and ingress gateway, as well as the ability to forward client information to workloads (client address via XFF).

## Details

API specification on the root level is split into **components** and **config**. The **components** field specifies the subset of [k8s component specification](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec) for `pilot` and `ingressGateway` Istio components. For now, it's **hpaSpec**, **strategy** and **resources**. The **config** parameter holds configuration of `profiles`, support for XFF, Gardener DNS names' configuration, tracing, and DNS proxying. 

The status is implemented per `kyma-operator`
 requirements.

## Istio CR example

```bash
apiVersion: operator.kyma-project.io/v1alpha1
kind: Istio
metadata:
spec:
  components: -> subset of k8s component spec
    pilot:
      k8s:
        hpaSpec: 
          maxReplicas: 5
          minReplicas: 2
        strategy:
          rollingUpdate:
            maxSurge: 50%
            maxUnavailable: "0"
        resources:
          limits:
            cpu: 250m
            memory: 384Mi
          requests:
            cpu: 10m
            memory: 128Mi
    ingressGateway:
      k8s:
        hpaSpec: 
          maxReplicas: 5
          minReplicas: 2
        strategy:
          rollingUpdate:
            maxSurge: 50%
            maxUnavailable: "0"
        resources:
          limits:
            cpu: 2000m
            memory: 1024Mi
          requests:
            cpu: 100m
            memory: 128Mi
  config: -> abstraction
    profile: evaluation | production
    xffHops/numTrustedProxies: ...
    gardenerDnsNames:
    - "*.mydomain.com"
    - ...
    tracing:
      ...
    dnsProxying:
      enabled: true | false
      autoAllocate: true | false
status:
  state: Ready | Processing | Error | Deleting
```

## Future plans

The proposal does not address the resource settings for sidecars. Currently, this configuration can be applied through the IstioOperator CR, but Istio does not support restarting sidecars. It is still being considered whether Istio Manager should be responsible for managing sidecars.