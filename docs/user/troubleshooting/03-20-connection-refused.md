# Connection Refused Errors

Follow this troubleshooting guide if a connection to a workload in the service mesh fails before you receive an HTTP response. A refused or reset connection can come from mTLS or from a NetworkPolicy, and the fix differs in each case.

## Identify the Origin of the Error

A connection reset by mTLS and a connection blocked by a NetworkPolicy fail in different ways, and the error text tells them apart. The decisive signal is whether the connection reaches the Envoy proxy at all.

Send a request to the failing workload and inspect the error. Compare your output with the following two cases:

- **mTLS resets the connection:** The connection fails with `Connection reset by peer` or `GOAWAY` when a service without a sidecar connects to a service with a sidecar. The connection reached the Envoy proxy, but the proxy rejected it because the peer presented no valid TLS certificate. See [mTLS Resets the Connection](#mtls-resets-the-connection).
- **A NetworkPolicy blocks the connection:** The connection fails with `Connection refused` or a connection timeout, with no `503` response and no `server: istio-envoy` header. A NetworkPolicy dropped the packets at the network layer, below Istio, so the connection never reached the Envoy proxy. See [A NetworkPolicy Blocks the Connection](#a-networkpolicy-blocks-the-connection).

> ### NOTE:
> Use the error text and the absence of a mesh response to decide the origin. An mTLS reset reaches the Envoy proxy and is actively reset, whereas a NetworkPolicy drop never reaches the proxy, so the connection is refused or times out without any `server: istio-envoy` header.

## mTLS Resets the Connection

### Symptom

You get either the `Connection reset by peer` response or the `GOAWAY` response when you attempt to establish the connection between a service without a sidecar and a service with a sidecar.

### Cause

By default, mutual TLS (mTLS) is enabled in the service mesh. As a result, every element of the service mesh must have an Istio sidecar with a valid TLS certificate to allow communication.

### Solution

- To add a service without a sidecar to the allowlist and disable mTLS traffic for it, create a DestinationRule resource. See [DestinationRule](https://istio.io/docs/reference/config/networking/destination-rule/).
- To allow connections between a service without a sidecar and a service with a sidecar, create a PeerAuthentication resource in the `PERMISSIVE` mode. See [Peer Authentication](https://istio.io/latest/docs/reference/config/security/peer_authentication/).

## A NetworkPolicy Blocks the Connection

### Symptom

A connection fails with `Connection refused` or a connection timeout, and no `503` response and no `server: istio-envoy` header appear. A NetworkPolicy operates at the network layer, below Istio. If it drops the packets, the request never reaches the Envoy proxy, so the connection times out or is refused instead of returning a mesh response.

### Cause

By default, a Pod accepts traffic from any source. When you create a NetworkPolicy that selects a Pod, that Pod rejects all traffic that the policy does not explicitly allow. Two cases break a workload in the service mesh:

- **A NetworkPolicy blocks traffic to or from the application.** The policy selects an istio-enabled workload and does not allow the traffic that you send. The request to the application times out, although Istio is configured correctly.
- **A NetworkPolicy blocks a sidecar from reaching the Istio control plane.** The policy restricts egress from an istio-enabled workload and does not allow traffic to `istiod` on the xDS port `15012`. The sidecar cannot fetch its configuration, so its routing becomes stale, and new connections fail.

### Solution

Follow these steps to find out whether a NetworkPolicy causes the failure and how to fix it.

1. Check whether any NetworkPolicy resources are present. If the cluster has none, a NetworkPolicy is not the cause:

    ```bash
    kubectl get networkpolicies --all-namespaces
    ```
    ```
    NAMESPACE     NAME                       POD-SELECTOR            AGE
    kube-system   gardener.cloud--deny-all   <none>                 4h
    ```

    The `POD-SELECTOR` column shows which Pods each policy selects. An empty selector, shown as `<none>`, selects every Pod in the policy's namespace, including istio-enabled workloads.

2. Find out whether a policy selects an istio-enabled workload or an Istio component. First, confirm that the affected Pod runs a sidecar. A Pod with a sidecar reports two containers in the `READY` column, such as `2/2`:

    ```bash
    kubectl get pods --namespace {YOUR_NAMESPACE}
    ```

    Then check the policies in the Pod's namespace and the `istio-system` namespace, and compare each policy's `spec.podSelector` with the Pod's labels:

    ```bash
    kubectl get networkpolicies --namespace {YOUR_NAMESPACE} --output yaml
    kubectl get pods --namespace {YOUR_NAMESPACE} --selector {YOUR_SELECTOR} --show-labels
    ```

    A policy applies to the Pod when its `spec.podSelector` matches the Pod's labels, or when the selector is empty.

3. Check whether the sidecar still receives its configuration from `istiod`. If a policy blocks egress to the control plane, the proxy cannot synchronize:

    ```bash
    istioctl proxy-status
    ```
    ```
    NAME                                     CLUSTER        ISTIOD                    VERSION   SUBSCRIBED TYPES
    httpbin-545d7f5848-4n2x8.netpol-demo     Kubernetes     istiod-54db84ffcf-mmspn   1.31.1    4 (CDS,LDS,EDS,RDS)
    ```

    A healthy proxy appears in the list, reports an `ISTIOD` instance, and subscribes to the `CDS`, `LDS`, `EDS`, and `RDS` configuration types. If the proxy is missing from the output, reports `STALE`, or does not subscribe to all types, it cannot reach `istiod`. Verify that a NetworkPolicy allows egress from the workload to `istiod` on the following ports:

    - `15012` for the xDS configuration and certificate signing
    - `15014` for control plane monitoring

4. Adjust the NetworkPolicy resources. A NetworkPolicy only allows traffic, so you add rules until the required traffic is permitted:

    - Allow ingress to the application from its clients on the application port.
    - Allow egress from istio-enabled workloads to `istiod` in the `istio-system` namespace on ports `15012` and `15014`.
    - Allow egress to the cluster DNS service on port `53`, so the sidecar can resolve service names.

As a best practice, follow these recommendations when you use NetworkPolicy resources with the service mesh:

- Start from a known baseline. Apply a default-deny policy, then add explicit allow rules for the traffic that you expect, instead of denying individual flows.
- Keep the control plane reachable. Always allow egress to `istiod` on ports `15012` and `15014` from every istio-enabled workload, and allow DNS egress on port `53`.
- Scope policies with precise selectors. A policy with an empty `spec.podSelector` selects every Pod in the namespace, which can block Istio components unintentionally.
- Verify the effect after each change. Run `istioctl proxy-status` to confirm that every sidecar still synchronizes with `istiod`.

> ### Note:
> A NetworkPolicy is enforced by the cluster's network plugin (CNI). If the plugin does not enforce NetworkPolicy resources, the policies have no effect on traffic. For more information about NetworkPolicy resources, see [Network Policies](../00-50-network-policies.md).
