# **trustDomain**

A **trustDomain** defines the security boundary of a service mesh.
It becomes part of each proxy identity and is embedded into the Subject Alternative Name (SAN) of workload certificates. 
Istio uses SPIFFE identities in the following format:

```
spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>
```

By default, Istio uses the **trustDomain** `cluster.local`. You can override it to align identities across clusters or to match organizational naming standards.

## Configure trustDomain in the Istio custom resource

Set the **trustDomain** under `spec.config.trustDomain` in the Istio CR. If omitted, the default value `cluster.local` is used.

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

## Where does the **trustDomain** appear?

Some of the key places where the **trustDomain** is used include:

- **Istio proxy identity certificate**: The **trustDomain** is included in the SAN of the certificate issued to every istio proxy in the mesh.
- **Authorization policies**: When defining AuthorizationPolicies, you can either use the default `cluster.local`
or your custom **trustDomain** in the SPIFFE IDs to specify which workloads the policy applies to.

## Multi-mesh seamless mTLS communication

When communicating between multiple meshes, in case a [shared Root CA is plugged in](../00-25-plug-in-istio-ca.md) to issue proxy certificates in both meshes,
Gateway with **tls.mode: ISTIO_MUTUAL** can be used to seamlessly allow mTLS communication between the two meshes without the need to set up `mTLS` on the Gateway manually.
In this case, the **trustDomain** can be used to differentiate the source of the request in the upstream mesh (from inside the mesh vs from another mesh),
as the SPIFFE ID in the client certificate will include the **trustDomain** of the originating mesh.

In this case an example Gateway configuration in the target mesh can be as simple as:

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

And in the source mesh, a ServiceEntry and DestinationRule can be set up to direct traffic to the target mesh with mTLS:

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

## Migration considerations

Changing the **trustDomain** is a sensitive operation. Once you update it existing workloads may fail authentication until they receive new certificates.
The default `cluster.local` value is treated as a special case that is compatible with custom **trustDomains**,
meaning that setting the **trustDomain** from `cluster.local` to a new, custom value should not cause authentication failures
for existing workloads until they are restarted and receive new certificates with the updated **trustDomain**.
With that said, it is still recommended to plan the change carefully and perform it during a maintenance window to minimize disruption.
For detailed migration guide, 
follow the [Istio trust domain migration guidance](https://istio.io/latest/docs/tasks/security/authorization/authz-td-migration)
to plan and execute the change with minimal disruption.

## References

- Istio **trustDomain** migration: https://istio.io/latest/docs/tasks/security/authorization/authz-td-migration/
- Istio mesh config reference: https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/
- SPIFFE **trustDomain** spec: https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE-ID.md#21-trust-domain
