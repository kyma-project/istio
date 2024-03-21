# Istio Basic Diagnostics

If you are experiencing issues with the cluster, you can follow the steps below to troubleshoot the problem before you contact us. This guide is designed to help you identify the root cause of the problem which might not necessarily be related to the Istio itself but rather to the misconfiguration of the Istio custom resources (CRs) or other cluster resources.

## Network connectivity

1. Check if Istio CR is in Warning state, if yes check for the warning message and conditions. It might be helpful at the beginning of the investigation.
2. Check [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) first. Try to find custom configuration which may affect the connectivity.
3. Check for the following custom resources applied on the cluster. Misconfigured might interrupt the cluster's traffic:
    - [`DestinationRule`](https://istio.io/latest/docs/reference/config/networking/destination-rule/) - misconfigured might interrupt the ability to access workloads.
    - [`PeerAuthentication`](https://istio.io/latest/docs/reference/config/security/peer_authentication/) - misconfigured might cause issues in the traffic between workloads.
    - [`Gateway`](https://istio.io/latest/docs/reference/config/networking/gateway/) - misconfigured might cause issues with access to the cluster's workloads from the outside.
    - [`AuthorizationPolicy`](https://istio.io/latest/docs/reference/config/security/authorization-policy/) - misconfigured might cause access control issues.
    - [`RequestAuthentication`](https://istio.io/latest/docs/reference/config/security/request_authentication/) - misconfigured might cause authentication issues.
4. You can use`istioctl` to get more insights into the Envoy Proxy. It can be used to check the configuration, logs etc. You can use command `istioctl proxy-config log deployment/${NAME_OF_DEPLOYMENT} -n istio-system --level "debug"`. For more information, see [istioctl](https://istio.io/latest/docs/reference/commands/istioctl/).
5. Check response flags: DC (Downstream client terminated connection), UC (Upstream terminated connection) is out of scope for Istio team, since it relates to client or workload application behaviour.

## Sidecar injection

1. Check if Istio CR is in Warning state, if yes check for the warning message. It might be helpful at the beginning of the investigation.
2. Check if the label `istio-injection=enabled` is present on the namespace where the application is deployed. If not, the sidecar will not be injected.
3. Make sure the pod does not have `hostNetwork: true` in the spec. If it does, the sidecar will not be injected.
4. Make sure pod does not overwrite the `istio-injection` label with `disabled` value. If it does, the sidecar will not be injected.

## If none of the above help, feel free to contact us. When you do, please do the following:
- Before creating a new issue, check for already existing issues and verify if the issues are related.
- Try to provide as much information as possible, including the steps to reproduce the issue, logs, and any other relevant information.
