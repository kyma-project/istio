# Istio v1alpha1 proposal

## Summary 

This document proposes  istio-manager API exposed to users. Initial version covers hpa and resources configuration for Istio components such as istiod and ingress gateway. This version also covers configuration needed to forward client information to workloads (client address via xff).

## Details

API specification on root level was split into `components` and `config`. `components` holds subset of [k8s component specyfication](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec) for `pilot` and `ingressGateway` Istio components. For this moment it's `hpaSpec`, `strategy` and `resources`. `config` holds `profiles` configuration, support for xff, gardener DNS names configuration, tracing and DNS proxying. 

Status will be implemented per `kyma-operator` requirements.

## Istio CR example

```
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

Proposal does not cover resource settings for sidecars. This configuration is applied via IstioOperator CR but Istio does not support restarting sidecars. It is still under consideration wheater `istio-manager` should be responsible for managing sidecars.