# Network Policies
Learn about the network policies for the Istio module, enable the network policy support, and allow egress traffic to your workloads.

## Overview
To enable secure-by-default practices, the Istio module allows creation of network policies in the `istio-system` and `kyma-system` namespaces.
These policies restrict traffic to and from Istio components, ensuring that only necessary baseline communication is allowed.
The policies make sure that in case a deny-by-default policy is applied at the cluster or namespace level,
the Istio module's components can still function properly.

## Enable Network Policy Support
To enable support for network policies, set the flag `networkPoliciesEnabled: true` in the Istio custom resource. This setting is disabled by default.

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  networkPoliciesEnabled: true
```

When the flag changes, Istio components are restarted, terminating existing TCP connections and enforcing the policies immediately.

## Network Policies Applied by the Istio Module
When enabled, the module applies network policies in the `istio-system` and `kyma-system` namespaces. This table lists the network policies applied when support is enabled and the traffic they allow:

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

All module-managed policies are labeled with:

- `kyma-project.io/module: istio`
- `kyma-project.io/managed-by: kyma`

## Networking Diagram

The following diagram illustrates the allowed traffic flows between Istio components and user workloads when network policy support is enabled.

In the diagram, network policies are illustrated as the resources through which allowed traffic flows. In reality, a network policy is a custom resource that configures which traffic is allowed or denied, while the actual filtering is performed by the Istio module's components.

![Istio Module NetworkPolicies](../../assets/network-policies-istio.svg)

### Enable Egress from `istio-ingressgateway` to Your Workloads

Because the egress traffic from `istio-ingressgateway` to user workloads is restricted by default, you must take additional steps to allow traffic to your applications.

To allow egress traffic from `istio-ingressgateway` to your workloads,
add this label to the Pods that should be reachable from `istio-ingressgateway`:

- `networking.kyma-project.io/from-ingressgateway: allowed`

See the following example workload template snippet:

```yaml
spec:
  template:
    metadata:
      labels:
        networking.kyma-project.io/from-ingressgateway: allowed
```

### [Optional] Apply a Deny-By-Default Policy

To isolate a workload's namespace with a deny-by-default policy, make sure to allow ingress from `istio-ingressgateway` in that policy.
See an example NetworkPolicy resource allowing ingress from `istio-ingressgateway` to the workload:

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

In case you are also using `egressgateway` (for details, see [Sending Requests Using Istio Egress Gateway](../tutorials/01-50-send-requests-using-egress.md))
and want to allow traffic from your workloads to `egressgateway`, add this label to the Pods: `networking.kyma-project.io/to-egressgateway: allowed`.
