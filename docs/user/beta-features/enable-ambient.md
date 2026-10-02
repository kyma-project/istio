# Enable Ambient Mode

Enable ambient mode when you want to adopt Istio's sidecarless service mesh without injecting a proxy container into every workload Pod.

By default, the Istio module uses the sidecar model, where an Envoy proxy is injected into each workload Pod to handle traffic encryption, observability, and policy enforcement. Ambient mode replaces this per-Pod sidecar with a shared, per-node Layer 4 proxy called ztunnel and an optional per-namespace Layer 7 proxy called waypoint.

When you set `enableAmbient` to `true` in the `istio-features` ConfigMap, the Istio module applies the following configuration changes:

- **ztunnel DaemonSet** – The ztunnel component is deployed as a DaemonSet. It handles encrypted mTLS communication between workloads at Layer 4 without requiring a sidecar in each Pod.
- **CNI ambient support** – The Istio CNI node agent is configured with `cni.ambient.enabled: true` so that it can redirect Pod traffic into the ztunnel on each node.
- **Pilot ambient support** – The `PILOT_ENABLE_AMBIENT` environment variable is set to `true` on istiod, enabling the control plane to manage ambient-mode workloads.
- **HBONE transport** – All Envoy proxies and ztunnel instances are configured with `ISTIO_META_ENABLE_HBONE=true` via mesh config `defaultConfig.proxyMetadata`, enabling the HTTP-Based Overlay Network Encapsulation (HBONE) tunneling protocol used by ambient mode.

> [!NOTE]
> Ambient mode can also be enabled through the Istio CR field `spec.experimental.enableAmbient`. The `enableAmbient` feature flag in the `istio-features` ConfigMap is an **OR** condition: ambient mode is active when either the ConfigMap flag **or** the CR field is set to `true`.

> [!WARNING]
> Support for ambient mode in the Istio module is experimental and may be changed or removed in any future release without prior notice.
