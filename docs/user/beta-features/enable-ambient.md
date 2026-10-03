# Enable Ambient Mode

Enable ambient mode when you want to adopt Istio's sidecarless service mesh without injecting a proxy container into every workload Pod.

By default, the Istio module uses the sidecar model, where an Envoy proxy is injected into each workload Pod to handle traffic encryption, observability, and policy enforcement. Ambient mode replaces this per-Pod sidecar with a shared, per-node Layer 4 proxy called ztunnel and an optional per-namespace Layer 7 proxy called waypoint. For more information, see [Sidecar or ambient?](https://istio.io/latest/docs/overview/dataplane-modes/).

When you set `enableAmbient` to `true` in the `istio-features` ConfigMap, the Istio module applies the following configuration changes:

- **ztunnel DaemonSet** – The ztunnel component is deployed as a DaemonSet. It handles encrypted mTLS communication between workloads at Layer 4 without requiring a sidecar in each Pod.
- **CNI ambient support** – The Istio CNI node agent is configured with `cni.ambient.enabled: true` so that it can redirect Pod traffic into the ztunnel on each node.
- **Pilot ambient support** – The `PILOT_ENABLE_AMBIENT` environment variable is set to `true` on istiod, enabling the control plane to manage ambient-mode workloads.
- **HBONE transport** – All Envoy proxies and ztunnel instances are configured with `ISTIO_META_ENABLE_HBONE=true` via mesh config `defaultConfig.proxyMetadata`, enabling the HTTP-Based Overlay Network Encapsulation (HBONE) tunneling protocol used by ambient mode.

> [!NOTE]
> If you use the experimental version of the Istio module, you can also enable the ambient mode in the **spec.experimental.enableAmbient** field in the Istio CR's field. The **enableAmbient** feature flag in the `istio-features` ConfigMap is an **OR** condition: ambient mode is active when either the ConfigMap flag **or** the CR field is set to `true`.

> [!CAUTION]
> Support for ambient mode in the Istio module is a beta feature that may be changed or removed in any future release without prior notice.
