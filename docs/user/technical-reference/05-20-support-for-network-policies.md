# Support for NetworkPolicies

## Overview
To enable secure-by-default practices, the Istio module allows creation of NetworkPolicies in `istio-system` and `kyma-system` namespaces.
These policies restrict traffic to and from Istio components, ensuring that only baseline necessary communication is allowed.
The policies make sure that in case a deny-by-default policy is applied at the cluster or namespace level,
Istio module components can still function properly.

## Enable NetworkPolicy support
Enable support by setting the flag in the Istio custom resource. This setting is disabled by default.

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  enableModuleNetworkPolicies: true
```

When the flag changes, Istio components are restarted so that existing TCP connections are terminated and the policies are enforced immediately.

## What the module applies
When enabled, the module applies NetworkPolicies in `istio-system` and `kyma-system` that allow:

- DNS egress (TCP/UDP 53) for Istio components.
- Kubernetes API server access (TCP 443) for `istio-controller-manager`, `istiod`, and `istio-cni-node`.
- Control-plane communication between `istio-ingressgateway` and `istiod` on TCP 15012.
- Ingress to `istiod` for XDS (15012), metrics (15014), and webhook calls (15017).
- Ingress to `istio-ingressgateway` for external traffic (8080/8443) and operational ports (15008, 15020, 15021, 15090).
- Ingress to `istio-egressgateway` from user workloads that are labeled with `networking.kyma-project.io/to-egressgateway: allowed`.
- Egress from `istiod` to JWKS endpoints on TCP 80/443 necessary for JWT validation in context of RequestAuthentication policies.
- Egress from `istio-ingressgateway` to user workloads that are labeled with `networking.kyma-project.io/from-ingressgateway: allowed`.
- Egress from `istio-egressgateway` to all destinations, as the egress traffic from `istio-egressgateway` should be controlled by Istio resources.

All module-managed policies are labeled with:

- `kyma-project.io/module: istio`
- `kyma-project.io/managed-by: kyma`

These resources should not be modified by users, as they are automatically updated by the module and any manual changes will be overwritten.

This table summarizes the allowed traffic when NetworkPolicy support is enabled:

| Component                | Namespace    | Port  | Protocol  | Direction | Purpose                                                                                       |
|--------------------------|--------------|-------|-----------|-----------|-----------------------------------------------------------------------------------------------|
| istio-controller-manager | kyma-system  | 53    | UDP/TCP   | egress    | DNS resolution                                                                                |
| istio-controller-manager | kyma-system  | 443   | TCP       | egress    | Kubernetes API server access                                                                  |
| istiod                   | istio-system | 53    | UDP/TCP   | egress    | DNS resolution                                                                                |
| istiod                   | istio-system | 80    | TCP       | egress    | Access to external JWKS endpoints for JWT validation (HTTP)                                   |
| istiod                   | istio-system | 443   | TCP       | egress    | Access  to external JWKS endpoints for JWT validation (HTTPS) / Kubernetes API server access  |
| istiod                   | istio-system | 15012 | TCP/gRPC  | ingress   | XDS config distribution to sidecars and gateways                                              |
| istiod                   | istio-system | 15014 | TCP/HTTP  | ingress   | Control plane metrics (Prometheus scrape)                                                     |
| istiod                   | istio-system | 15017 | TCP/HTTPS | ingress   | Webhook endpoint (defaulting/mutation/admission)                                              |
| istio-egressgateway      | istio-system | *     | UDP/TCP   | egress    | All outbound traffic from egress is allowed, as the configuration is done via Istio resources |
| istio-egressgateway      | istio-system | *     | UDP/TCP   | ingress   | Traffic from labeled user Pods (`networking.kyma-project.io/to-egressgateway: allowed`)       |
| istio-ingressgateway     | istio-system | *     | TCP       | egress    | Traffic to labeled user Pods (`networking.kyma-project.io/from-ingressgateway: allowed`)      |
| istio-ingressgateway     | istio-system | 53    | UDP/TCP   | egress    | DNS resolution                                                                                |
| istio-ingressgateway     | istio-system | 8080  | TCP       | ingress   | HTTP traffic inbound to cluster                                                               |
| istio-ingressgateway     | istio-system | 8443  | TCP       | ingress   | HTTPS traffic inbound to cluster                                                              |
| istio-ingressgateway     | istio-system | 15008 | TCP       | ingress   | HBONE mTLS tunnel (Ambient mode)                                                              |
| istio-ingressgateway     | istio-system | 15012 | TCP/gRPC  | egress    | Request XDS config from istiod                                                                |
| istio-ingressgateway     | istio-system | 15020 | TCP/HTTP  | ingress   | Merged Prometheus metrics                                                                     |
| istio-ingressgateway     | istio-system | 15021 | TCP/HTTP  | ingress   | Health check endpoint                                                                         |
| istio-ingressgateway     | istio-system | 15090 | TCP/HTTP  | ingress   | Envoy Prometheus telemetry                                                                    |
| istio-cni-node           | istio-system | 53    | UDP/TCP   | egress    | DNS resolution                                                                                |
| istio-cni-node           | istio-system | 443   | TCP       | egress    | Kubernetes API server access                                                                  |

## Networking diagram

The following diagram illustrates the allowed traffic flows between Istio components and user workloads when NetworkPolicy support is enabled.
Requests that are allowed by NetworkPolicies flow through the NetworkPolicy.

![Istio Module NetworkPolicies](../../assets/network-policies-istio.svg)

## What users must do to keep connectivity
Because the egress traffic from `istio-ingressgateway` to user workloads is restricted by default, you need to take additional steps to allow traffic to your applications.

### Enable egress from `istio-ingressgateway` to user workloads

To allow egress traffic from `istio-ingressgateway` to your workloads,
add this label to the Pods that should be reachable from `istio-ingressgateway`:

- `networking.kyma-project.io/from-ingressgateway: allowed`

Example workload template snippet:

```yaml
spec:
  template:
    metadata:
      labels:
        networking.kyma-project.io/from-ingressgateway: allowed
```

### [Optional] Apply a deny-by-default policy

In case the workload namespace should be isolated with a deny-by-default policy, make sure to allow ingress from `istio-ingressgateway` in that policy as well.
Example NetworkPolicy allowing ingress from `istio-ingressgateway` to the workload:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
    name: allow-ingress-from-istio-ingressgateway
    namespace: my-namespace
spec:
    podSelector:
      matchLabels:
        app: my-app
    policyTypes:
    - Ingress
    ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: istio-system
        podSelector:
          matchLabels:
            istio: ingressgateway
      ports:
        - protocol: TCP
          port: 8080 # The targetPort of the application container
```
