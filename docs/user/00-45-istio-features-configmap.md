# Alpha Istio Features
Use the `istio-features` ConfigMap to enable or disable alpha Istio features that are not exposed in the Istio custom resource (CR) due to stability, compliance, or security concerns.

>[!WARNING]
> The `istio-features` ConfigMap may contain features that are:
> - **Not ready for production use** – Features may be unstable or behave unexpectedly under load.
> - **Not recommended due to compliance or security requirements** – Some features may reduce the security posture of your cluster or conflict with your organization's policies.
> - **Subject to removal** – Features exposed through this ConfigMap may be changed or removed in any future release without prior notice.
>
> Istio module authors cannot guarantee full support where these features are enabled.
> Use this ConfigMap only if you fully understand the implications of each feature you enable.

## Overview

The `istio-features` ConfigMap allows you to control non-default Istio module behaviors that are intentionally not exposed in the Istio CR. When the ConfigMap is created, updated, or deleted in the `kyma-system` namespace, the Istio module controller detects the change and reconciles accordingly. Feature flags are defined as a JSON object under the **features** key.

## Creating the `istio-features` ConfigMap

To enable a feature, create the ConfigMap in the `kyma-system` namespace with the desired feature flags set:

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

To disable all feature flags or reset to defaults, delete the ConfigMap or remove the `features` key:

```bash
kubectl delete configmap istio-features -n kyma-system
```

## Available Features

### **disableCni**

**Type:** `boolean`
**Default:** `false`

When set to `true`, the Istio CNI node agent is not deployed. Instead, Istio uses an `istio-init` init container to configure network traffic interception in each Pod.

>[!WARNING]
> Disabling Istio CNI has significant security implications. Carefully read the risks below before enabling this flag.

#### Security Risks of Disabling Istio CNI

By default, the Istio module deploys the [Istio CNI node agent](https://istio.io/latest/docs/setup/additional-setup/cni/) as a DaemonSet. The CNI plugin configures each Pod's network namespace without requiring elevated privileges in application containers or init containers.

When you disable CNI (`disableCni: true`), Istio falls back to using an `istio-init` init container injected into every meshed Pod. This init container requires the `NET_ADMIN` and `NET_RAW` Linux capabilities to set up `iptables` rules that redirect traffic to the `istio-proxy` sidecar.

This introduces the following risks:

- **Elevated privileges in application Pods** – The `istio-init` init container requires `NET_ADMIN` and `NET_RAW` capabilities. These capabilities allow the container to modify network configuration within its network namespace and may be prohibited by your cluster's `PodSecurity` admission policy or security scanning tools.
- **Bypass risk** – Any container in the Pod that runs before `istio-init` completes, or any container that also holds `NET_ADMIN`/`NET_RAW` capabilities, could potentially modify or bypass the `iptables` rules that enforce traffic interception. With Istio CNI, this concern is eliminated because interception is set up by a privileged node-level agent before the Pod's containers start.
- **Increased attack surface on nodes** – While the `istio-init` container only affects its own network namespace, having `NET_ADMIN`-capable init containers increases the overall attack surface compared to the CNI-based approach, where privilege escalation is confined to the dedicated CNI DaemonSet.


## Feature Flags

| Flag         | Type      | Default | Description                                                                                                                                                                                     |
|--------------|-----------|---------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **disableCni** | `boolean` | `false` | When `true`, disables the Istio CNI node agent and falls back to the `istio-init` init container approach. See [Security Risks of Disabling Istio CNI](#security-risks-of-disabling-istio-cni). |
