# Enable Dual Stack

Enable dual-stack support when your cluster is provisioned with both IPv4 and IPv6 addresses and you need services in the mesh to be reachable over both protocols.

To enable dual-stack support (IPv4 and IPv6) for the Istio service mesh, set `enableDualStack` to `true` in the `istio-features` ConfigMap. This configures Istio Pilot and all gateways to use `RequireDualStack` IP family policy and propagates the `ISTIO_DUAL_STACK` environment variable to all Envoy proxies.

When you enable this feature and the cluster load balancer is dual-stack, it has the following effects:

- Istio Pilot is configured with `ISTIO_DUAL_STACK=true`.
- Istio Pilot, ingress gateway, and egress gateway services use `ipFamilyPolicy: RequireDualStack`.
- Sidecar proxies receive `ISTIO_DUAL_STACK=true` via mesh config `defaultConfig.proxyMetadata`.

> [!NOTE]
> This flag only takes effect when the cluster load balancer is also configured for dual-stack. The `kyma-provisioning-info` ConfigMap in the `kyma-system` namespace must exist and contain `dualStackIPEnabled: true` under `networkDetails`. If the `kyma-provisioning-info` ConfigMap is absent or the field is `false`, enabling dual-stack in the `istio-features` ConfigMap has no effect.

> [!CAUTION]
> Enabling dual-stack is a one-way operation and cannot be reversed. Once `enableDualStack` is set to `true` and reconciliation runs, setting the flag back to `false` or deleting the ConfigMap doesn't revert the Services to single-stack.
