# Disable Istio CNI
Use this feature when your workloads use a sandboxed runtime such as gVisor, where Istio CNI is incompatible with the sandbox's network stack.

When you set `disableCni` to `true` in the `istio-features` ConfigMap, the Istio CNI node agent is not deployed. Instead, Istio uses an `istio-init` init container to configure network traffic interception in each Pod. This init container requires the `NET_ADMIN` and `NET_RAW` Linux capabilities to set up `iptables` rules that redirect traffic to the `istio-proxy` sidecar.

> [!CAUTION]
> Disabling Istio CNI has significant security implications. Consider the following risks before enabling this flag:
> - **Elevated privileges in application Pods:** The `istio-init` init container requires `NET_ADMIN` and `NET_RAW` capabilities. These capabilities allow the container to modify network configuration within its network namespace and may be prohibited by your cluster's `PodSecurity` admission policy or security scanning tools.
> - **Bypass risk:** Any container in the Pod that runs before `istio-init` completes, or any container that also holds `NET_ADMIN`/`NET_RAW` capabilities, could potentially modify or bypass the `iptables` rules that enforce traffic interception. With Istio CNI, this concern is eliminated because interception is set up by a privileged node-level agent before the Pod's containers start.
> - **Increased attack surface on nodes:** While the `istio-init` container only affects its own network namespace, having `NET_ADMIN`-capable init containers increases the overall attack surface compared to the CNI-based approach, where privilege escalation is confined to the dedicated CNI DaemonSet.