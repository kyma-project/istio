# DNS Proxying

Learn how DNS proxying in Istio improves DNS performance, enables ServiceEntry resolution, and solves routing issues for external TCP services.

## Overview

DNS proxying intercepts DNS requests from applications and resolves them locally at the Istio sidecar proxy (or ztunnel in ambient mode). The proxy maintains a local mapping of domain names to IP addresses. If a domain can be resolved locally, the proxy responds immediately. Otherwise, it forwards the request upstream following the standard `/etc/resolv.conf` DNS configuration.

## Problems DNS Proxying Solves

### Reduced Load on Cluster DNS

Without DNS proxying, every DNS query from workloads goes to kube-dns. With DNS proxying enabled, the sidecar resolves known service addresses locally, reducing traffic to the cluster DNS server.

### ServiceEntry Resolution

When you define a [ServiceEntry](https://istio.io/latest/docs/reference/config/networking/service-entry/) with a custom hostname (for example, `address.internal`), cluster DNS does not know about it. Without DNS proxying, applications cannot resolve these addresses. DNS proxying allows the sidecar to resolve ServiceEntry hostnames directly.

### External TCP Services on the Same Port

Istio routes TCP traffic based on destination IP and port only. Unlike HTTP traffic, which includes a Host header, TCP has no additional metadata for routing decisions.

When multiple external TCP services share the same port (for example, two databases on port 3306), Istio cannot distinguish between them without unique IP addresses. The sidecar creates a single listener on `0.0.0.0:{port}` and forwards traffic to only one destination.

DNS proxying solves this by auto-allocating virtual IPs (VIPs) from the `240.240.0.0/16` range to each ServiceEntry. This gives each external service a unique address, enabling the sidecar to route traffic correctly.

## How to Enable DNS Proxying

### Sidecar Mode

DNS proxying is disabled by default in sidecar mode. You can enable it cluster-wide or per workload.

#### Cluster-Wide Configuration

Set **enableDNSProxying** to `true` in the Istio custom resource (CR) configuration:

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    enableDNSProxying: true
```

This sets `ISTIO_META_DNS_CAPTURE` to `true` for all sidecar proxies in the mesh.

#### Per-Workload Configuration

Add the `proxy.istio.io/config` annotation to enable DNS proxying for a specific Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        proxy.istio.io/config: |
          proxyMetadata:
            ISTIO_META_DNS_CAPTURE: "true"
    spec:
      containers:
      - name: my-app
        image: my-app:latest
```

## Auto-Allocation of Virtual IPs

When DNS proxying is enabled and a ServiceEntry does not specify an explicit IP address in the `addresses` field, Istio auto-allocates a virtual IP from the reserved Class E range (`240.240.0.0/16`).

For example, a ServiceEntry like this:

```yaml
apiVersion: networking.istio.io/v1
kind: ServiceEntry
metadata:
  name: external-db
spec:
  hosts:
  - db.example.com
  ports:
  - number: 3306
    name: tcp
    protocol: TCP
  resolution: DNS
```

Results in DNS queries for `db.example.com` returning an auto-allocated IP like `240.240.0.1` instead of the actual external IP. The sidecar then routes traffic for `240.240.0.1:3306` to the resolved backend.

To opt out of auto-allocation for a specific ServiceEntry, add the following label:

```yaml
metadata:
  labels:
    networking.istio.io/enable-autoallocate-ip: "false"
```

> **NOTE:** Auto-allocation does not work for wildcard hosts (for example, `*.example.com`).

## Consequences

### Benefits

- **Performance**: Reduced DNS query latency and lower load on kube-dns.
- **ServiceEntry support**: Applications can resolve hostnames defined in ServiceEntry resources.
- **TCP routing**: Multiple external TCP services on the same port work correctly with auto-allocated VIPs.
- **Mesh visibility**: Istio gains visibility and control over DNS resolution.

### Considerations

- **Non-routable IPs**: Auto-allocated addresses use the `240.240.0.0/16` range. Applications that validate or log IP addresses may see unexpected values.
- **Proxy complexity**: The sidecar takes on DNS resolution responsibilities, slightly increasing resource usage.
- **No wildcard support**: Auto-allocation does not apply to ServiceEntry resources with wildcard hosts.

## Related Information

- [Istio DNS Proxy documentation](https://istio.io/latest/docs/ops/configuration/traffic-management/dns-proxy/)
- [DNS Configuration in Istio](https://istio.io/latest/docs/ops/configuration/traffic-management/dns/)
- [Egress Traffic Control](https://istio.io/latest/docs/tasks/traffic-management/egress/egress-control/)
