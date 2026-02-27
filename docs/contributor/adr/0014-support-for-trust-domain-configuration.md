# Support for Trust Domain Configuration

## Status

Accepted

## Context

In Istio's security model, a trust domain is a fundamental concept that represents the security boundary of a service mesh. It corresponds to the trust root of a system and is used in workload identity certificates (SPIFFE certificates). The trust domain is encoded in the Subject Alternative Name (SAN) field of the X.509 certificates issued to workloads within the mesh.

By default, Istio uses `cluster.local` as the trust domain. However, there are scenarios where users need to customize the trust domain. For example, in multi-cluster deployments, where multiple clusters need to trust each other, a shared trust domain can facilitate seamless service-to-service authentication across cluster boundaries without requiring a complex federation setup.

The trust domain affects:
- The SPIFFE identity format: `spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>`
- Certificate validation and authentication between services
- Authorization policies that reference identities

Currently, the `cluster.local` trust domain is hardcoded in the Istio operator configuration, and users have no way to customize it through the Istio custom resource.

## Decision

We will add support for configuring the trust domain through the Istio custom resource. The implementation will include the following aspects:

1. **Add the trustDomain field**: The field will be added to the **Config** struct in the Istio CR specification with the following properties:
   - Type: `string` (optional)
   - Pattern validation: must conform to the SPIFFE specification for trust domain names
   - Length constraints: 1-255 characters
   - Default value: `cluster.local` (when not specified)

2. **Merge the configuration** into the IstioOperator's **trustDomain** field during reconciliation, allowing the user-specified value to override the default.

3. **Validation**: The field will be validated by Kubernetes API server through CRD validation rules to ensure it meets the SPIFFE specification requirements for trust domain names.

4. **Backward compatibility**: If the field is not specified, the existing default behavior (`cluster.local`) will be maintained, ensuring no breaking changes for existing deployments.

5. **Downtime**: During the trust domain change, workloads may experience temporary authentication failures until new certificates are issued and propagated.