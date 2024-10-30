# Support for **egressGateway** Configuration

## Status

Accepted

## Context

To support cluster configurations that require control of outbound traffic
besides routing of incoming traffic, the Istio module needs to support `egressGateway`.
This Istio component allows for intercepting traffic coming from in-mesh Pods to 
targets outside of the cluster scope. This allows users to perform tasks that include
monitoring and securing access to outbound resources.

## Decision

We will introduce configuration for `egressGateway` Istio component in the
Istio CustomResource (CR). The API will allow to conditionally enable
the `egressGateway` as well as configuration for [KubernetesResourcesSpec(k8s)](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).

## Consequences

The Istio CR API will be extended with configuration related to `egressGateway`. The `components` section will now include an additional `egressGateway` field containing a boolean `enabled` flag as well as configuration for `KubernetesResourcesSpec(k8s)`. The `k8s` configuration will have the exact same structure as for `ingressGateway`.

This results in the folowing Go structure:

```go
// EgressGateway defines configuration for Istio egressGateway
type EgressGateway struct {
	// +kubebuilder:validation:Optional
	K8s *KubernetesResourcesConfig `json:"k8s"`
        // +kubebuilder:validation:Optional
        Enabled *bool
}
```

The `KubernetesResourcesConfig` struct is [already defined](https://github.com/kyma-project/istio/blob/04890425c106ffd564d4c209994f99b4e692f9ec/api/v1alpha2/istio_structs.go#L37) in the Istio controller.

An example Istio CR could look as follows:

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  components:
    egressGateway:
      enabled: true
      k8s:
        hpaSpec:
          maxReplicas: 10
          minReplicas: 3
```

## Default Values

By default, the `egressGateway` component will be disabled.

The `egressGateway` component must have default values set for when the user does not set up `k8s` values. Since the component will most likely have a high load, the best course of action would be to propose the exact values after executing performance tests and ensure that the availability does not drop when the component is in place.

The baseline values could be the same as we use for Istio Ingress Gateway:

```yaml
# Full installation
hpaSpec:
  maxReplicas: 10
  metrics:
  - resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 80
    type: Resource
  minReplicas: 3
resources:
  limits:
    cpu: 2000m
    memory: 1024Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

```yaml
# Light installation
hpaSpec:
  maxReplicas: 1
  minReplicas: 1
resources:
  limits:
    cpu: 1000m
    memory: 1024Mi
  requests:
    cpu: 10m
    memory: 32Mi
```

