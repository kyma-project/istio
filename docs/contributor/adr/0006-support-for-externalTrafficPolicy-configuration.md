# Support for **externalTrafficPolicy** Configuration

## Status

Accepted

## Context
There is a need to support the configuration of **externalTrafficPolicy** as the default value of `Cluster` does not allow for forwarding the correct value of **X-Forwarded-For** headers. This is a common requirement for applications that need to know the original client IP address.

## Considerations

### Should Istio Ingress Gateway Be Deployed as a DaemonSet, or as Another Configuration that Would Make Sure at Least one Pod Runs per Node?

Reference: [Network Load Balancer](https://istio.io/latest/docs/tasks/security/authorization/authz-ingress/#network)

For production Deployments, it is strongly recommended to deploy an Ingress Gateway Pod to multiple nodes if you enable `externalTrafficPolicy: Local`. Otherwise, this creates a situation where only nodes with an active Ingress Gateway Pod are able to accept and distribute incoming NLB traffic to the rest of the cluster, creating potential ingress traffic bottlenecks and reduced internal load balancing capability or even complete loss of ingress traffic to the cluster if the subset of nodes with Ingress Gateway Pods goes down. See Source IP for Services with `Type=NodePort` for more information.

**Answer**: Configure PodAntiAffinity, which will spread Pods evenly among users.

## Decision
To allow for this configuration, the **externalTrafficPolicy** should be configurable in the Istio custom resource. The default value should be `Cluster` to maintain the current behavior. Additionally, configuration for the value `Local` should be supported.

Configuring the field **gatewayExternalTrafficPolicy** to `Local` will additionally configure Ingress Gateway PodAntiAffinity to make sure that Pods are spread evenly across nodes.

## Consequences
Istio CustomResourceDefinition will be extended with an additional configuration field of **externalTrafficPolicy** that will allow for configuration of the **externalTrafficPolicy** setting in Istio Ingress Gateway. The field will be an optional configuration with the default value of `Cluster`.

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
    gatewayExternalTrafficPolicy: Local
    numTrustedProxies: 1
```
This will set the value of **externalTrafficPolicy** to `Local`.

3.
```yaml
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
spec:
  config:
    gatewayExternalTrafficPolicy: Cluster
    numTrustedProxies: 1
```
This will set the value of **externalTrafficPolicy** to `Cluster`.
