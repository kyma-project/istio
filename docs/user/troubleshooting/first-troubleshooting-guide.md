# Istio troubleshooting guide

If you are experiencing issues with the cluster, you can follow the steps below to troubleshoot the problem before you contact us. This guide is designed to help you identify the root cause of the problem which might not necessarily be related to the Istio itself but rather to the misconfiguration of the Istio Custom Resources or other network related resources.

## Try to go through the following steps:

1. When an Istio problem is reported for a Kyma module, SRE should first check whether it also occurs for other modules. If the problem only occurs with a specific module, the team that owns that module should start investigating first before involving us.
2. For connection issues, before forwarding it to Istio module team check [Calico NetworkPolicies](https://docs.tigera.io/calico/latest/network-policy/get-started/calico-policy/calico-network-policy) first. Try to find custom configuration which may affect the connectivity.
3. Check if the IstioCR is in warning status, if yes it indicates that the user action is required. Details can be found on the IstioCR.
4. Check for the following custom resources applied on the cluster. Misconfigured might interrupt the cluster's traffic:
    - [`DestinationRule`](https://istio.io/latest/docs/reference/config/networking/destination-rule/) - misconfigured might interrupt ability to access the workloads.
    - [`PeerAuthentication`](https://istio.io/latest/docs/reference/config/security/peer_authentication/) - misconfigured might cause issues in the traffic between workloads.
    - [`Gateway`](https://istio.io/latest/docs/reference/config/networking/gateway/) - misconfigured might cause issues with access to the cluster's workloads from the outside.
    - [`AuthorizationPolicy`](https://istio.io/latest/docs/reference/config/security/authorization-policy/) - misconfigured might cause access control issues.
    - [`RequestAuthentication`](https://istio.io/latest/docs/reference/config/security/request_authentication/) - misconfigured might cause authentication issues.
5. You can use`istioctl` to get more insights into the Envoy Proxy. It can be used to check the configuration, logs etc. You can use command `istioctl proxy-config log deployment/${NAME_OF_DEPLOYMENT} -n istio-system --level "debug"`. For more information, see [istioctl](https://istio.io/latest/docs/reference/commands/istioctl/).
6. Check response flags: DC (Downstream client terminated connection), UC (Upstream terminated connection) is out of scope for Istio team, since it relates to client or workload application behaviour.

## If none of the above steps help, feel free to contact us. When you do, please do the following:
- Before creating a new issue for a cluster, check for already existing issues for that cluster and verify if the issues are related.
- Try to provide as much information as possible, including the steps to reproduce the issue, logs, and any other relevant information.
