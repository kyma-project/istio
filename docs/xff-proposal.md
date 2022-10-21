# Istio v1alpha1 proposal

## Summary 

This document proposes  istio-manager API exposed to users. Initial version covers hpa and resources configuration for Istio components such as istiod, ingress gateway and sidecars. This version also covers configuration needed to forward client information to workloads (client address via xff).

## Details

API specification on root level was split into `controlPlane` and `dataPlane` as described in Istio overview. `controlPlane` holds resources and hpa configuration for `istiod` component and general configuration called `meshConfig`. For this moment `meshConfig` will expose configuration to define `numTrustedProxies`, which is required for xff. `dataPlane` configuration contains resources and hpa configuration for `ingressGateway` component.

Status will be implemented per `kyma-operator` requirements.

## Istio CR example

```
apiVersion: operator.kyma-project.io/v1alpha1
kind: Istio
metadata:
spec:
  controlplane:
    istiod:
      deployment:
        hpa: 
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
      meshConfig:
        gatewayTopology:
          numTrustedProxies: <VALUE>
  dataplane:
    ingressGateway:
      deployment:
        hpa: 
          maxReplicas: 5
          minReplicas: 2
          metrics:
          - resource:
              name: cpu
              targetAverageUtilization: 80
            type: Resource
          - resource:
              name: memory
              targetAverageUtilization: 80
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
status:
  state: Ready | Processing | Error | Deleting
```

## Future plans

In the future `meshConfig` specification can be enhanced with tracing specific configuration by defining `extensionProvider`. `ingressGateway` specification can be enhanced to enable users to define gardener domain annotations simplifying use of custom domain to expose workload.

Proposal does not cover resource settings for sidecars. This configuration is applied via IstioOperator CR but Istio does not support restarting sidecars. It is still under consideration wheater `istio-manager` should be responsible for managing sidecars.

Future changes like tracing, sidecar resources or gardener domain annotations will be covered in following proposals.