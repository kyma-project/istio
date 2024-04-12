# Support for `externalTrafficPolicy` configuration

## Status

| Date       | Status   |
|------------|----------|
| 11.04.2024 | Proposed |

## Context
There is a need to support configuration of the `externalTrafficPolicy` as the default value of `Cluster` does not allow for forwarding correct value of `X-Forwarded-For` headers. This is a common requirement for applications that need to know the original client IP address.

## Decision
To allow for this configuration, the `externalTrafficPolicy` should be configurable in the `Istio` Custom Resource. The default value should be `Cluster` to maintain the current behavior. Additionally, configuration for the value of `Local` should be supported.

## Consequences
Istio Custom Resource Definition will be extended with additional configuration field of `externalTrafficPolicy` that will allow for configuration of the `externalTrafficPolicy` setting in Istio Ingress Gateway. The field will be an optional configuration with default value of `Cluster`.

### Sample configuration:

1.
```yaml
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
```
This will use the default value of `Cluster` in Istio Ingress Gateway.

2.
```yaml
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
spec:
  config:
    gatewayexternalTrafficPolicy: Local
```
This will set the value of `externalTrafficPolicy` to `Local`.

3.
```yaml
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
spec:
  config:
    gatewayexternalTrafficPolicy: Cluster
```
This will set the value of `externalTrafficPolicy` to `Cluster`.

## Considerations

### Should Istio Ingress Gateway be deployed as a DaemonSet?

Reference: `https://istio.io/latest/docs/tasks/security/authorization/authz-ingress/#network`
> For production deployments, it is strongly recommended to deploy an ingress gateway pod to multiple nodes if you enable externalTrafficPolicy: Local. Otherwise, this creates a situation where only nodes with an active ingress gateway pod will be able to accept and distribute incoming NLB traffic to the rest of the cluster, creating potential ingress traffic bottlenecks and reduced internal load balancing capability, or even complete loss of ingress traffic to the cluster if the subset of nodes with ingress gateway pods go down. See Source IP for Services with Type=NodePort for more information.
