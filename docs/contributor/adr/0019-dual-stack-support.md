# Dual-Stack Support for Istio on Dual-Stack Infrastructure

## Status

Accepted

## Context

### IPv6 and dual-stack fundamentals

Dual-stack networking means a cluster runs both IPv4 and IPv6 simultaneously. On such infrastructure, Kubernetes assigns both an IPv4 and an IPv6 address to every node and pod. Services can be configured to receive both a ClusterIP v4 and a ClusterIP v6.

For Istio to work correctly on a dual-stack cluster, it must be explicitly told that dual-stack is in use. Without this, the `istio-proxy` sidecar intercepts only IPv4 traffic. IPv6 traffic bypasses the sidecar entirely, which has two serious consequences:

1. **Security**: all Istio policies (AuthorizationPolicy, PeerAuthentication, mTLS) are silently bypassed for IPv6 traffic. A workload becomes reachable both inside and outside the mesh at the same time — inside for IPv4 connections, outside for IPv6 ones — making policy enforcement unpredictable.
2. **Incomplete load balancer configuration**: without dual-stack awareness, Istio does not configure its Ingress Gateway Service or the cloud Load Balancer to accept IPv6 connections.

Enabling dual-stack in Istio involves a number of coordinated changes across the Istio installation itself (control plane and data plane configuration), the Ingress Gateway Service, and — depending on the cloud provider — the Load Balancer setup and Proxy Protocol handling. The exact set of changes is an implementation detail that may evolve as the feature matures.

Istio upstream classifies dual-stack support as a **beta** feature (see [Istio feature status](https://istio.io/latest/docs/releases/feature-stages/)). It is functional and we have validated it on AWS and GCP, but Istio upstream does not yet consider it production-ready.

### Evolution of the implementation

The feature went through three iterations before reaching the current design:

**Version 1 — explicit opt-in via Istio CR (experimental channel only)**

The first implementation added an `enableDualStack: true` flag under `spec.experimental` in the Istio CR. Users on a dual-stack cluster had to set this flag manually. The feature was only available in the experimental channel of the Istio module.

**Version 2 — auto-detection via `kyma-provisioning-info` (experimental channel only)**

The explicit flag was removed in favour of auto-detection: the controller read `dualStackIPEnabled: true` from the `kyma-provisioning-info` ConfigMap in `kyma-system` and enabled dual-stack automatically when the infrastructure supported it. This version was still restricted to the experimental channel.

**Version 3 — opt-in via `istio-features` ConfigMap (all channels)**

To allow users outside the experimental channel to test the feature, dual-stack was made available in all channels via the `istio-features` ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: istio-features
  namespace: kyma-system
data:
  features: |
    {
      "enableDualStack": true
    }
```

This is the current state. The flag is intended as a temporary testing mechanism, not a permanent user-facing switch.

## Decision

All viable alternatives to full Istio dual-stack on dual-stack infrastructure were evaluated and found to be either insecure or incomplete (see _Considered alternatives_). The conclusion is that **full Istio dual-stack is the only correct configuration for a dual-stack cluster**.

Given that, exposing this as a user-configurable option does not make sense:

- Allowing users to run Istio single-stack on a dual-stack cluster would leave a permanent security hole — IPv6 traffic bypasses the mesh entirely.
- Switching between single-stack and dual-stack after installation is a complex migration: it requires reprovisioning the Load Balancer, which changes its IP address and therefore invalidates DNS records and any external references to the cluster.

The only safe and operationally sound approach is to **always align Istio's stack configuration with the underlying infrastructure from the very beginning**, without requiring or offering a user-facing toggle.

The Istio module will therefore automatically enable full dual-stack support on dual-stack infrastructure. The exact detection and configuration mechanism is a technical implementation detail to be decided separately.

Although Istio dual-stack is still a beta feature upstream, we accept this in order to:
- ensure users on dual-stack clusters are protected by the mesh from the start,
- give users early exposure to the feature so they are prepared when it reaches GA.

## Consequences

### What becomes easier

- Users on dual-stack clusters are automatically protected: IPv6 traffic is intercepted by the service mesh without any manual steps.
- There is no risk of misconfiguration: the stack mode is always consistent with the infrastructure.
- Users gain early familiarity with Istio dual-stack behavior before it reaches GA upstream.

### What becomes more complex

- The Istio module controller must detect the infrastructure stack mode and apply it consistently.
- Because Istio dual-stack is a **beta feature upstream**, behavior may change between Istio minor versions. The team must watch upstream release notes for dual-stack-related changes.

## Known obstacles

### Automated testing of dual-stack

A dedicated dual-stack end-to-end test suite has been implemented in the API Gateway module. The tests spin up a real dual-stack cluster on demand, but the GitHub Actions runners used in CI are single-stack and cannot make IPv6 connections to that cluster. This means the dual-stack test results cannot be fully verified from standard CI workers.

## Considered alternatives

### Option 1: Run Istio single-stack on dual-stack infrastructure

**Rejected — critical security issue.**

When a cluster runs in dual-stack mode, Kubernetes assigns IPv6 addresses to both nodes and pods. If Istio is not aware of dual-stack, its `istio-proxy` only intercepts IPv4 traffic. Any client that connects to a workload via IPv6 (either directly to the pod IPv6 address or via an IPv6 ClusterIP) bypasses the sidecar entirely.

This was verified: connecting to a protected endpoint via IPv6 returned `200 OK` instead of `403 Forbidden`, and the Envoy headers were absent from the response. The fact that AuthorizationPolicies were not enforced was an immediate no-go.

### Option 2: Run Istio single-stack on single-stack Kubernetes, with a dual-stack Load Balancer (IPv6-to-IPv4 translation)

**Rejected — vendor-specific, no IPv6 egress, and transition mechanisms are impractical.**

Some cloud load balancers can be configured to accept IPv6 connections from outside and forward them to IPv4 targets. This means the cluster can receive IPv6 ingress traffic even though Kubernetes itself only has IPv4. With infrastructure-level dual-stack enabled in the Gardener shoot, the VPC and subnets get IPv6 CIDR blocks, and the Load Balancer Controller can provision a dual-stack NLB without assigning IPv6 to nodes or pods.

This avoids the Istio security bypass (since pods have no IPv6 address, there is nothing to connect to via IPv6 directly), and it was the most promising alternative for offering IPv6 ingress without accepting a beta feature.

However, this approach has several fundamental limitations:

**Vendor-specific, no standard**: IPv6-to-IPv4 translation at the load balancer level is not defined by any official standard and relies on vendor-specific implementation. This makes it a non-starter for a multi-cloud solution.

**No IPv6 egress**: pods have no IPv6 address and the node has no IPv6 stack. This was verified in practice — IPv6 egress from pods fails immediately at the most basic level with `Network unreachable`, before any transition mechanism even comes into play.

Even setting aside this hard failure, any IPv6 transition mechanism that could theoretically bridge this gap would carry a high operational cost that should not be underestimated. These solutions are inherently fragile — troubleshooting networking issues that span multiple layers requires coordinated debugging across multiple teams and projects (such as Gardener and Cloud Manager), each with visibility into only their own layer, making root cause analysis extremely complex. On top of that, components that manipulate low-level networking inside Kubernetes are difficult to upgrade without downtime and interact non-trivially with the CNI; infrastructure-level changes are also cloud-vendor dependent.

It is also worth noting that the absence of an "alpha" or "beta" label on an alternative tool does not automatically mean it is better supported than Istio dual-stack. Istio, despite the beta label on this specific feature, has a large, active community behind it.

This trade-off (vendor-specific IPv6 ingress only, no IPv6 egress, with highly complex workarounds) was deemed worse than enabling Istio dual-stack, so the approach was abandoned.

### Option 3: Route IPv6 egress traffic through the Istio egress gateway

**Rejected — only covers in-mesh workloads.**

An alternative considered for the egress problem was to route IPv6 outbound traffic through the Istio egress gateway, which could be configured with IPv6 connectivity. However, this only works for workloads that are part of the Istio service mesh. Any workload running outside the mesh — which is a legitimate and common configuration — would still have no IPv6 egress. This makes the solution incomplete and inconsistent from a user perspective, so it was not pursued.

### Option 4: Route IPv6 egress traffic through an HTTP/S or SOCKS5 proxy

**Rejected — high operational cost and limited applicability.**

Another egress alternative considered was running an HTTP/S or SOCKS5 proxy with IPv6 connectivity that workloads could use to reach IPv6 destinations. This was dropped early without testing because the operational cost was deemed prohibitive:

- Every client application must be explicitly configured to use the proxy, and configuration differs across languages, frameworks, and runtimes.
- The proxy operates at layer 7 (HTTP) or at most at the TCP session level (SOCKS5), so it cannot transparently handle all traffic. Raw TCP connections and non-HTTP protocols are either unsupported or require additional work.
- Debugging connectivity issues through a proxy is significantly harder than debugging direct connections.

This approach would place a significant burden on application developers and operators for a partial solution, so it was not pursued.

### Option 5: User-managed Gateway via Kubernetes Gateway API

**Rejected — does not solve the underlying problems.**

This idea was considered as a way to let users manage their own dual-stack gateway, avoiding the need to change the default Istio Ingress Gateway or the default Kyma gateway. However, it does not solve any of the fundamental issues:

- Istio still needs to be switched to dual-stack mode at the installation level, otherwise IPv6 traffic bypasses the mesh and the security hole remains.
- A dual-stack load balancer cannot be provisioned if the underlying network infrastructure does not support it — the load balancer controller cannot obtain an IPv6 address without infrastructure-level dual-stack enabled.
- Users would need to manage cloud-provider annotations manually. Every annotation change triggers a load balancer reprovisioning, changing its IP address, and load balancer resources can leak during this process.
- `APIRule` resources do not work with Kubernetes Gateways — they reference Istio Gateways and use VirtualServices. The Istio Gateway that is autogenerated under the hood when a Kubernetes Gateway is created has an unpredictable internal name, and even when referenced correctly, routes are not attached and the setup does not work in practice.
