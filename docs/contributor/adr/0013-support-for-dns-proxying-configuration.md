# Support for DNS Proxying Configuration

## Status

Proposed

## Context 
<!--What is the issue that we're seeing that is motivating this decision or change?-->

### Problem Statement
1. Applications accessing external services without stable IP addresses rely on DNS, but standard resolution doesn't provide Istio with enough context for proper routing.
2. Istio cannot distinguish between multiple TCP services operating on the same port without stable IP addresses. Unlike HTTP requests (which use Host headers), TCP traffic can only be routed based on destination IP and port, leading to conflicts when multiple `ServiceEntries` share the same port.
3. If the client was unable to resolve the DNS request, the request would terminate before Istio receives it. This means that if a request is sent to a hostname which is known to Istio (for example, by a `ServiceEntry`) but not to the DNS server, the request will fail.

Without DNS proxying:
- DNS requests bypass the Istio proxy and go directly to upstream DNS servers
- Istio lacks visibility into DNS resolution, limiting its ability to route traffic accurately 
- ServiceEntry addresses unknown to the cluster DNS server cause request failures, even if Istio knows about them
- High load on the cluster's Kubernetes DNS server


## Decision 
<!--What is the change that we're proposing and/or doing?-->

### How It Works

With DNS proxying enabled:
1. All DNS requests from applications are intercepted and redirected to the sidecar proxy (or ztunnel in ambient mode)
2. The proxy maintains a local mapping of domain names to IP addresses
3. If the proxy can resolve the request locally, it responds directly without contacting upstream DNS servers
4. Otherwise, the request is forwarded upstream following the standard `/etc/resolv.conf` configuration
5. The proxy can auto-allocate non-routable virtual IPs (VIPs) to distinguish between multiple external TCP services on the same port as long as they do not use a wildcard host in `ServiceEntry`

Configuration of DNS Proxying behavior by mode:
  - Sidecar mode: Field enables or disables DNS proxying for sidecar proxy. 
  - Ambient mode : Field doesn't affect any configuration of DNS Proxying for ztunnel proxy.
      - Rationale: DNS proxying is always enabled in ambient mode (Istio ≥1.25)
      - Can only be disabled per-workload via `ambient.istio.io/dns-capture=false` annotation
      - Setting this field does not have any effect and may cause confusion
    
### Scope

We will add support for configuring DNS proxying cluster-wide through the Istio custom resource for Sidecar mode. The implementation includes:

1. **Add the `enableDNSProxying` field**: 
    - Location: **Config** struct in the Istio CR specification 
    - Type: `*bool` (optional)
    - Default: none (uses mode-specific defaults when unset; Ambient- enabled by default, Sidecar- disabled by default)
    - UI Integration: Configurable and displayed in Kyma Dashboard

2. **Configuration Merging**
   - User value merges into the IstioOperator `defaultConfig.proxyMetadata.ISTIO_META_DNS_CAPTURE` setting
   - Conversion: `boolean` (Istio CR) → `string` (IstioOperator: "true"/"false") during reconciliation
   - User-specified value overrides Sidecar mode defaults

3. **Validation**: 
   - Boolean validation via CRD

4. **Backward compatibility**: 
   - When `enableDNSProxying` is unset:
     - Ambient mode: DNS proxying enabled (default)
     - Sidecar mode: DNS proxying disabled (default)
   - No breaking changes for existing deployments


## Consequences
<!--What becomes easier or more difficult to do because of this change?-->

Exercising control over the application’s DNS resolution allows Istio to accurately identify the target service to which traffic is bound, and enhance the overall security, routing, and telemetry posture in Istio within and across clusters. As of this change:
- The load on the cluster’s Kubernetes DNS server drops.
- The issue of distinguishing between multiple external TCP services without VIPs on the same port is solved as long as they do not use a wildcard host.
- Applications can resolve ServiceEntry addresses even if the cluster DNS server doesn't know about them.
- Istio gains visibility and control over DNS resolution within the mesh.
- The sidecar/ztunnel proxy takes on DNS resolution duties, adding complexity.
- Sidecar mode gains the ability to configure DNS proxying cluster-wide. In ambient mode (Istio ≥1.25), DNS proxying is always enabled and cannot be disabled via the `ISTIO_META_DNS_CAPTURE` setting.