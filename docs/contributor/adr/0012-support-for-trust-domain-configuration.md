# Support for Trust Domain Configuration

## Status

Proposed //@TODO change to Accepted upon approval

## Context

In Istio's security model, a trust domain is a fundamental concept that represents the security boundary of a service mesh. It corresponds to the trust root of a system and is used in workload identity certificates (SPIFFE certificates). The trust domain is encoded in the Subject Alternative Name (SAN) field of the X.509 certificates issued to workloads within the mesh.

By default, Istio uses `cluster.local` as the trust domain. However, there are scenarios where users need to customize the trust domain:
 **Multi-cluster deployments**: When running multiple clusters that need to trust each other, a shared trust domain enables seamless service-to-service authentication across cluster boundaries without requiring complex federation setup.

The trust domain affects:
- The SPIFFE identity format: `spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>`
- Certificate validation and authentication between services
- Authorization policies that reference identities

Currently, the trust domain is hardcoded to `cluster.local` in the Istio operator configuration, and users have no way to customize it through the Istio custom resource.

## Decision

We will add support for configuring the trust domain through the Istio custom resource. The implementation will:

1. **Add a `trustDomain` field** to the `Config` struct in the Istio CR specification with the following properties:
   - Type: `string` (optional)
   - Pattern validation: `^[a-z0-9._-]+$` (lowercase alphanumeric characters, dots, underscores, and hyphens)
   - Length constraints: 1-255 characters
   - Default value: `cluster.local` (when not specified)

2. **Merge the configuration** into the IstioOperator's `trustDomain` field during reconciliation, allowing the user-specified value to override the default.

3. **Validation**: The field will be validated by Kubernetes API server through CRD validation rules to ensure it meets the SPIFFE specification requirements for trust domain names.

4. **Backward compatibility**: If the field is not specified, the existing default behavior (`cluster.local`) will be maintained, ensuring no breaking changes for existing deployments.

[//]: # ()
[//]: # (## Consequences)

[//]: # ()
[//]: # (### Positive)

[//]: # ()
[//]: # (- **Flexibility**: Users can now customize trust domains to match their organizational requirements and deployment topologies.)

[//]: # (- **Multi-cluster support**: Enables better support for multi-cluster and multi-mesh scenarios where distinct trust domains are needed.)

[//]: # (- **Standards compliance**: Allows users to implement trust domains that comply with their security policies and naming conventions.)

[//]: # (- **Backward compatible**: Existing deployments continue to work without any changes.)

[//]: # ()
[//]: # (### Negative)

[//]: # ()
[//]: # (- **Configuration complexity**: Users need to understand the implications of changing trust domains, especially in running clusters.)

[//]: # (- **Migration risk**: Changing the trust domain in an existing cluster will cause certificate re-issuance and may temporarily affect service-to-service authentication until all certificates are renewed.)

[//]: # (- **Cross-domain communication**: If not configured properly in multi-cluster scenarios, services may fail to authenticate with each other.)

[//]: # ()
[//]: # (### Considerations)

[//]: # ()
[//]: # (- **No runtime updates**: Changing the trust domain requires careful planning. It's recommended to set this value during initial installation. Changing it on a running cluster will trigger certificate rotation.)

[//]: # (- **Documentation needs**: Clear documentation should be provided on:)

[//]: # (  - When and why to customize the trust domain)

[//]: # (  - The impact of changing trust domains)

[//]: # (  - Best practices for multi-cluster deployments)

[//]: # (  - Migration procedures if trust domain needs to be changed)

[//]: # ()
[//]: # (### Sample Configuration)

[//]: # ()
[//]: # (1. Using the default trust domain &#40;implicit&#41;:)

[//]: # (```yaml)

[//]: # (apiVersion: operator.kyma-project.io/v1alpha2)

[//]: # (kind: Istio)

[//]: # (metadata:)

[//]: # (  name: default)

[//]: # (  namespace: kyma-system)

[//]: # (```)

[//]: # (This will use the default value of `cluster.local`.)

[//]: # ()
[//]: # (2. Specifying a custom trust domain:)

[//]: # (```yaml)

[//]: # (apiVersion: operator.kyma-project.io/v1alpha2)

[//]: # (kind: Istio)

[//]: # (metadata:)

[//]: # (  name: default)

[//]: # (  namespace: kyma-system)

[//]: # (spec:)

[//]: # (  config:)

[//]: # (    trustDomain: prod.company.internal)

[//]: # (```)

[//]: # (This will configure Istio to use `prod.company.internal` as the trust domain.)

[//]: # ()
[//]: # (This allows multiple clusters to share the same trust domain for seamless cross-cluster authentication.)

[//]: # ()
[//]: # (### Implementation Notes)

[//]: # ()
[//]: # (The trust domain configuration will be merged into the IstioOperator spec during the `mergeConfig` phase of reconciliation, following the same pattern used for other configuration fields like `numTrustedProxies` and `gatewayExternalTrafficPolicy`.)

[//]: # ()
