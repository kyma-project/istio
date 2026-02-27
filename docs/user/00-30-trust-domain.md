# Istio Trust Domain
Trust domain is a Istio concept used to align identities across multiple clusters, enable seamless multi-mesh mTLS communication with shared root CAs, and match organizational naming standards.

## What Is a Trust Domain?
A trust domain defines the security boundary of a service mesh.
It becomes part of each proxy identity and is embedded into the Subject Alternative Name (SAN) of workload certificates. 
Istio uses SPIFFE identities in the following format:

```
spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>
```

By default, Istio uses the trust domain `cluster.local`. You can override it to align identities across clusters or to match organizational naming standards.

## Configure a Trust Domain in the Istio Custom Resource

Set the trust domain under **spec.config.trustDomain** in the Istio CR. If omitted, the default value `cluster.local` is used.

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    trustDomain: "example.com"
```

## Where Does the Trust Domain Appear?

The trust domain is used in the following key contexts:

- Istio proxy identity certificate: The trust domain is included in the SAN of the certificate issued to every Istio proxy in the mesh.
- AuthorizationPolicies: When defining AuthorizationPolicies, you can either use the default value `cluster.local`
or your custom trust domain in the SPIFFE IDs to specify to which workloads the policy applies.

## Multi-Mesh Seamless mTLS Communication

If a [shared Root CA is plugged in](../00-25-plug-in-istio-ca.md) to issue proxy certificates in multiple meshes, you can use a Gateway with **tls.mode: ISTIO_MUTUAL** to seamlessly allow mTLS communication between them. In this case, you don't need to set up `mTLS` on the Gateway manually.

Additionally, you can use the trust domain to differentiate the source of requests in the upstream mesh (whether they come from within the same mesh or from another mesh). This is possible because the SPIFFE ID in the client certificate includes the trust domain of the originating mesh.

To configure this setup, you can use the following example Gateway configuration in the target mesh:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: multi-mesh-gateway
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: ISTIO_MUTUAL
    hosts:
    - "*.example-target.com"
```

In the source mesh, you can configure a ServiceEntry and DestinationRule to direct traffic to the target mesh with mTLS:

```yaml
apiVersion: networking.istio.io/v1
kind: ServiceEntry
metadata:
  name: external-httpbin
spec:
  hosts:
    - "httpbin.example-target.com"
  ports:
    - number: 80
      name: http
      protocol: HTTP
      targetPort: 443
  resolution: DNS
  location: MESH_EXTERNAL
---
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: external-httpbin-mtls
spec:
  host: "httpbin.example-target.com"
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
```

## Migration Considerations

Changing the trust domain is a sensitive operation. Once you update it, existing workloads may fail authentication until they receive new certificates.
The default `cluster.local` value is treated as a special case that is compatible with custom trust domains.
This means that setting the trust domain from `cluster.local` to a new, custom value doesn't cause authentication failures
for existing workloads until they are restarted and receive new certificates with the updated trust domain.
With that said, it is still recommended to plan the change carefully and perform it during a maintenance window to minimize disruption.
For a detailed migration guide, 
follow the [Istio trust domain migration guidance](https://istio.io/latest/docs/tasks/security/authorization/authz-td-migration)
to plan and execute the change with minimal disruption.

## Related Information

- [Istio trust domain migration](https://istio.io/latest/docs/tasks/security/authorization/authz-td-migration/)
- [Istio mesh configuration reference](https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/)
- [SPIFFE **trustDomain** specification](https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE-ID.md#21-trust-domain)
