# Istio Basic Diagnostics

If you are experiencing issues with the cluster, follow the steps below to troubleshoot the problem before you report it. This guide is designed to help you identify the root cause of the problem, which might not necessarily be related to Istio itself but rather to the misconfiguration of Istio custom resources (CRs) or other cluster resources.

## Network Connectivity

1. Verify the state of Istio CR. If it is in the `Warning` state, check the warning message and conditions. This might be helpful at the beginning of the investigation.
2. Verify that no [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) are affecting the connectivity by blocking traffic between Pods in the service mesh. To find all NetworkPolicies, run the command `kubectl get networkpolicies -A`.
3. The configuration of the following kinds of resources can affect the connectivity in the service mesh. Verify that those resources are configured as intended:
    - [`DestinationRule`](https://istio.io/latest/docs/reference/config/networking/destination-rule/)
    - [`PeerAuthentication`](https://istio.io/latest/docs/reference/config/security/peer_authentication/)
    - [`Gateway`](https://istio.io/latest/docs/reference/config/networking/gateway/)
    - [`AuthorizationPolicy`](https://istio.io/latest/docs/reference/config/security/authorization-policy/)
    - [`RequestAuthentication`](https://istio.io/latest/docs/reference/config/security/request_authentication/)
4. Use the command `istioctl analyze -A` to check for potential issues in the Istio configuration and see suggestions on how to fix them.
5. To enable the access logs of the Envoy proxy, follow the guide [Envoy Access Logs](https://istio.io/latest/docs/tasks/observability/logs/access-log/). In the access logs, you can find the field **response_flags**. The response flags DC (Downstream client terminated connection) and UC (Upstream terminated connection) are not within the scope of the Istio module, as they relate to the behavior of the client (DC) or the workload application (UC).

## Sidecar Injection

1. Verify if Istio CR is in the `Warning` state. If it is, check the warning message. It might be helpful at the beginning of the investigation.
2. Check if you correctly enabled sidecar injection. See the guide [Check If You Have Istio Sidecar Proxy Injection Enabled](https://kyma-project.io/#/istio/user/operation-guides/02-10-check-if-sidecar-injection-is-enabled?id=check-if-you-have-istio-sidecar-proxy-injection-enabled) for more information.
3. Make sure the Pod does not have `hostNetwork: true` in the spec. If it does, the sidecar will not be injected.

## Still Something Doesn't Work?
1. Check the [Official Istio Troubleshooting Guide](https://github.com/istio/istio/wiki/Troubleshooting-Istio).
2. Look for already existing issues in the [Istio module repository](https://github.com/kyma-project/istio/issues). If none of them is related to your problem, create a new issue.