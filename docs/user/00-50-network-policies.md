# Network Policies
Learn about the network policies for the Istio module, enable the network policy support, and allow egress traffic to your workloads.

## Context
To support secure-by-default configurations, the Istio module can create network policies in the `istio-system` and `kyma-system` namespaces. These policies restrict traffic to and from Istio components so that only the required baseline communication is allowed.
This helps ensure that the Istio module's components continue to function even when a deny-by-default policy is applied at the cluster or namespace level.

All module-managed policies use the following labels:

- `kyma-project.io/module: istio`
- `kyma-project.io/managed-by: kyma`

Do not modify these resources manually. The module updates them automatically and overwrites any manual changes.

## Networking Diagram

The following diagram illustrates the allowed traffic flows between Istio components and user workloads when network policy support is enabled.

In the diagram, network policies are shown as the resources that traffic passes through. In practice, a network policy is a custom resource that defines which traffic is allowed or denied, while the Istio module's components perform the actual filtering.

![Istio Module NetworkPolicies](../assets/network-policies-istio.svg)

## List of Network Policies

Review the network policies that the Istio module creates when network policy support is enabled.

<details>
<summary>Show table</summary>


| Component                  | Namespace      | Port    | Protocol  | Direction | Source/Destination                                                                                                                                                                                                                                            | Purpose                                                                                                       |
|----------------------------|----------------|---------|-----------|-----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------|
| `istio-controller-manager` | `kyma-system`  | `53`    | UDP/TCP   | Egress    | destination: *                                                                                                                                                                                                                                                | DNS resolution                                                                                                |
| `istio-controller-manager` | `kyma-system`  | `443`   | TCP       | Egress    | destination: *                                                                                                                                                                                                                                                | Kubernetes API server access                                                                                  |
| `istiod`                   | `istio-system` | `53`    | UDP/TCP   | Egress    | destination: *                                                                                                                                                                                                                                                | DNS resolution                                                                                                |
| `istiod`                   | `istio-system` | `80`    | TCP       | Egress    | destination: *                                                                                                                                                                                                                                                | Access to external JWKS endpoints for JWT validation (HTTP)                                                   |
| `istiod`                   | `istio-system` | `443`   | TCP       | Egress    | destination: *                                                                                                                                                                                                                                                | Access to external JWKS endpoints for JWT validation (HTTPS) / Kubernetes API server access                   |
| `istiod`                   | `istio-system` | `15012` | TCP/gRPC  | Ingress   | source (any of):</br> - podSelector `security.istio.io/tlsMode: istio` (any namespace) </br> - podSelector `istio: ingressgateway`                                                                                                                            | XDS config distribution to sidecars and gateways                                                              |
| `istiod`                   | `istio-system` | `15014` | TCP/HTTP  | Ingress   | source (any of):</br> - has label `kyma-project.io/module` (any namespace) </br> - podSelector `networking.kyma-project.io/istio-metrics: allowed` (any namespace) </br> - podSelector `networking.kyma-project.io/metrics-scraping: allowed` (any namespace) | Control plane metrics (Prometheus scrape)                                                                     |
| `istiod`                   | `istio-system` | `15017` | TCP/HTTPS | Ingress   | source: *                                                                                                                                                                                                                                                     | Webhook endpoint (defaulting/mutation/admission). It is genarally only accessed by the Kubernetes API server. |
| `istio-egressgateway`      | `istio-system` | `*`     | UDP/TCP   | Egress    |                                                                                                                                                                                                                                                               | All outbound traffic from egress is allowed, as the configuration is done via Istio resources                 |
| `istio-egressgateway`      | `istio-system` | `*`     | UDP/TCP   | Ingress   | source: podSelector `networking.kyma-project.io/to-egressgateway: allowed` (any namespace)                                                                                                                                                                    | Traffic from labeled user Pods (`networking.kyma-project.io/to-egressgateway: allowed`)                       |
| `istio-egressgateway`      | `istio-system` | `15020` | TCP/HTTP  | Ingress   | source (any of):</br> - has label `kyma-project.io/module` (any namespace) </br> - podSelector `networking.kyma-project.io/istio-metrics: allowed` (any namespace) </br> - podSelector `networking.kyma-project.io/metrics-scraping: allowed` (any namespace) | Merged Prometheus metrics                                                                                     |
| `istio-egressgateway`      | `istio-system` | `15021` | TCP/HTTP  | Ingress   | source (any of):</br> - has label `kyma-project.io/module` (any namespace) </br> - podSelector `networking.kyma-project.io/istio-metrics: allowed` (any namespace) </br> - podSelector `networking.kyma-project.io/metrics-scraping: allowed` (any namespace) | Health check endpoint                                                                                         |
| `istio-egressgateway`      | `istio-system` | `15090` | TCP/HTTP  | Ingress   | source (any of):</br> - has label `kyma-project.io/module` (any namespace) </br> - podSelector `networking.kyma-project.io/istio-metrics: allowed` (any namespace) </br> - podSelector `networking.kyma-project.io/metrics-scraping: allowed` (any namespace) | Envoy Prometheus telemetry                                                                                    |
| `istio-ingressgateway`     | `istio-system` | `*`     | TCP       | Egress    |                                                                                                                                                                                                                                                               | Traffic to labeled user Pods (`networking.kyma-project.io/from-ingressgateway: allowed`)                      |
| `istio-ingressgateway`     | `istio-system` | `*`     | TCP       | Egress    | destination: podselector `networking.kyma-project.io/from-ingressgateway: allowed` (any namespace)                                                                                                                                                            | Traffic to labeled user Pods (`networking.kyma-project.io/from-ingressgateway: allowed`)                      |
| `istio-ingressgateway`     | `istio-system` | `53`    | UDP/TCP   | Egress    | destination: *                                                                                                                                                                                                                                                | DNS resolution                                                                                                |
| `istio-ingressgateway`     | `istio-system` | `8080`  | TCP       | Ingress   | source: *                                                                                                                                                                                                                                                     | HTTP traffic inbound to cluster                                                                               |
| `istio-ingressgateway`     | `istio-system` | `8443`  | TCP       | Ingress   | source: *                                                                                                                                                                                                                                                     | HTTPS traffic inbound to cluster                                                                              |
| `istio-ingressgateway`     | `istio-system` | `15012` | TCP/gRPC  | Egress    | destination: podSelector `istio: pilot`                                                                                                                                                                                                                       | Request XDS config from istiod                                                                                |
| `istio-ingressgateway`     | `istio-system` | `15020` | TCP/HTTP  | Ingress   | source (any of):</br> - has label `kyma-project.io/module` (any namespace) </br> - podSelector `networking.kyma-project.io/istio-metrics: allowed` (any namespace) </br> - podSelector `networking.kyma-project.io/metrics-scraping: allowed` (any namespace) | Merged Prometheus metrics                                                                                     |
| `istio-ingressgateway`     | `istio-system` | `15021` | TCP/HTTP  | Ingress   | source (any of):</br> - has label `kyma-project.io/module` (any namespace) </br> - podSelector `networking.kyma-project.io/istio-metrics: allowed` (any namespace) </br> - podSelector `networking.kyma-project.io/metrics-scraping: allowed` (any namespace) | Health check endpoint                                                                                         |
| `istio-ingressgateway`     | `istio-system` | `15090` | TCP/HTTP  | Ingress   | source (any of):</br> - has label `kyma-project.io/module` (any namespace) </br> - podSelector `networking.kyma-project.io/istio-metrics: allowed` (any namespace) </br> - podSelector `networking.kyma-project.io/metrics-scraping: allowed` (any namespace) | Envoy Prometheus telemetry                                                                                    |
| `istio-cni-node`           | `istio-system` | `53`    | UDP/TCP   | Egress    | destination: *                                                                                                                                                                                                                                                | DNS resolution                                                                                                |
| `istio-cni-node`           | `istio-system` | `443`   | TCP       | Egress    | destination: *                                                                                                                                                                                                                                                | Kubernetes API server access                                                                                  |

</details>

## Procedure

1. Enable network policy support.

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

2. Enable egress from `istio-ingressgateway` to your workloads

    Because the egress traffic from `istio-ingressgateway` to user workloads is restricted by default, you must take additional steps to allow traffic to your applications.

    To allow egress traffic from `istio-ingressgateway` to your workloads,
    add this label to the Pods that should be reachable from `istio-ingressgateway`: `networking.kyma-project.io/from-ingressgateway: allowed`.

    See the following example workload template snippet:

    ```yaml
    spec:
      template:
        metadata:
          labels:
            networking.kyma-project.io/from-ingressgateway: allowed
    ```

3. Enable egress traffic from your workloads to `istio-egressgateway`

    In case you have `egressgateway` enabled and want to allow traffic from your workloads to `egressgateway`, add this label to the Pods: `networking.kyma-project.io/to-egressgateway: allowed`.


    See the following example workload template snippet:

    ```yaml
    spec:
      template:
        metadata:
          labels:
            networking.kyma-project.io/to-egressgateway: allowed
    ```

4. Enable access to metrics and health check endpoints.

    To allow access to the metrics and health check endpoints of `istio-ingressgateway` and `istio-egressgateway`, add either of these labels to the Pods that should be able to access those endpoints:
    - `networking.kyma-project.io/istio-metrics: allowed`
    - `networking.kyma-project.io/metrics-scraping: allowed`

    See the following example workload template snippet:

    ```yaml
    spec:
      template:
        metadata:
          labels:
            networking.kyma-project.io/istio-metrics: allowed
            networking.kyma-project.io/metrics-scraping: allowed
    ```

5. [Optional] Apply a deny-by-default policy.

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
