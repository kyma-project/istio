# Beta Istio Features

Enable or disable beta Istio features. These features aren't exposed in the Istio custom resource (CR) due to stability, compliance, or security concerns.

## Context

> [!CAUTION]
> Beta features are not ready for production use. Be aware of the following risks:
> - **Instability:** Beta features may be unstable or behave unexpectedly under load.
> - **Security and compliance concerns:** Beta features may reduce the security posture of your cluster or conflict with your organization's policies.
> - **Subject to removal:** Beta features may be changed or removed in any future release without prior notice.
>
> Beta features are not subject to SLAs, and full support is not guaranteed for clusters where they are enabled. Enable them only if you fully understand the impact of each feature.

To control beta Istio features, create the `istio-features` ConfigMap in the `kyma-system` namespace. Each beta feature is controlled by a feature flag, defined as a JSON object under the `features` key. When you create, update, or delete this ConfigMap, the Istio module controller detects the change and reconciles resources accordingly.

The following beta features are available in the Istio module:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `disableCni` | boolean | `false` | When `true`, disables the Istio CNI node agent and falls back to the `istio-init` init container approach. See [Disable Istio CNI](disable-istio-cni.md). |
| `enableControlPlaneVPA` | boolean | `false` | When `true`, creates VPA resources for Istio control plane components (istiod, gateways, CNI), managing memory only. Requires VPA CRD in the cluster. See [Enable Control Plane VPA](enable-control-plane-vpa.md). |
| `enableDualStack` | boolean | `false` | When `true`, enables dual-stack support (IPv4 and IPv6) for the Istio service mesh. Only takes effect when the cluster load balancer is also configured for dual stack (`kyma-provisioning-info` ConfigMap must be present with `dualStackIPEnabled: true`). Cannot be disabled once enabled. See [Enable Dual Stack](enable-dual-stack.md). |

## Procedure

- To enable a beta feature, create the `istio-features` ConfigMap with one or more feature flags set to `true`:

    ```bash
    kubectl apply -f - <<EOF
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: istio-features
      namespace: kyma-system
    data:
      features: |
        {
          "disableCni": true
        }
    EOF
    ```

- To disable all feature flags or reset to defaults, delete the ConfigMap or remove the `features` key:

    ```bash
    kubectl delete configmap istio-features -n kyma-system
    ```

    Once **enableDualStack** is set to `true`, deleting the ConfigMap doesn't revert the Services to single-stack.

