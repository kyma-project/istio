# VPA for Istio Control Plane Components

## Status
Proposed

## Context
This ADR addresses [kyma-project/istio#2011](https://github.com/kyma-project/istio/issues/2011).
The goal is to conditionally enable Vertical Pod Autoscaler (VPA) for Istio control plane components to support mesh scaling on large clusters. The VPA must coexist with the existing HPA without conflicts — HPA handles CPU-based horizontal scaling, while VPA optimizes memory resource allocation.

## Decision

### Feature Flag

A new boolean field `enableControlPlaneVPA` is added to the `IstioFeatures` struct in `internal/istiofeatures/features.go`:

```go
type IstioFeatures struct {
    DisableCni            bool `json:"disableCni"`
    EnableControlPlaneVPA bool `json:"enableControlPlaneVPA"`
}
```

Users enable it by setting the flag in the `istio-features` ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: istio-features
  namespace: kyma-system
data:
  features: |
    {
      "enableControlPlaneVPA": true
    }
```

### VPA Resources

When enabled, the operator creates VPA resources in `istio-system` for:

| Target               | Kind       | Container   | minAllowed memory | maxAllowed memory |
|----------------------|------------|-------------|-------------------|-------------------|
| istiod               | Deployment | discovery   | 512Mi             | 16Gi              |
| istio-ingressgateway | Deployment | istio-proxy | 128Mi             | 16Gi              |
| istio-egressgateway  | Deployment | istio-proxy | 128Mi             | 16Gi              |
| istio-cni-node       | DaemonSet  | install-cni | 512Mi             | 16Gi              |

All VPAs share these settings:
- `controlledResources: [memory]`
- `controlledValues: RequestsAndLimits`
- `updatePolicy.updateMode: InPlaceOrRecreate`
- `updatePolicy.minReplicas: 1`

### HPA Memory Metric Prevention

When `enableControlPlaneVPA` is true, the operator must ensure that no memory-based metric is present in the HPA for pilot or gateways. The upstream Istio chart template (`autoscale.yaml`) conditionally adds a memory metric if `values.pilot.memory.targetAverageUtilization` is set:

```yaml
{{- if .Values.memory.targetAverageUtilization }}
- type: Resource
  resource:
    name: memory
    ...
{{- end }}
```

To be defensive against future changes, during the IstioOperator merge step (`mergeResources` in `api/v1alpha2/istio_merge.go`), when `Features.EnableControlPlaneVPA` is true:

1. Remove any `memory` metric from `hpaSpec.metrics` for pilot, ingress gateway, and egress gateway components if present.
2. Ensure `values.pilot.memory` is not set (or explicitly set to empty) in the rendered IstioOperator values.

This guarantees HPA only scales on CPU while VPA manages memory.

### Backward Compatibility

- Clusters without VPA CRD: no VPA resources created (same guard as operator VPA).
- ConfigMap absent or feature not set: defaults to `false`, no VPAs created.
- Egress gateway disabled: In case egress-gateway is not deployed in the cluster VPA controller will ignore the VPA without an error.

## Consequences

- Clusters with VPA support and the feature enabled will have memory-optimized Istio control plane components without manual tuning.
- HPA continues to scale horizontally based on CPU; VPA adjusts memory vertically. No conflict between the two autoscalers.
- The feature is opt-in and alpha — users must explicitly enable it via ConfigMap.
