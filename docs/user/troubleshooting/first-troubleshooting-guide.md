# Istio Basic Diagnostics

If you are experiencing issues with the cluster, you can follow the steps below to troubleshoot the problem before you contact us. This guide is designed to help you identify the root cause of the problem which might not necessarily be related to the Istio itself but rather to the misconfiguration of the Istio custom resources (CRs) or other cluster resources.

## Network connectivity

1. Check if Istio CR is in Warning state, if yes check for the warning message and conditions. It might be helpful at the beginning of the investigation.
2. Verify that no [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) is affecting the connectivity by blocking traffic between pods in the mesh. You can find all network policies by running the command: `kubectl get networkpolicies -A`
3. The configuration of resources of the following kinds can affect the connectivity in the service mesh. Verify that those resources are configured as intended:
    - [`DestinationRule`](https://istio.io/latest/docs/reference/config/networking/destination-rule/)
    - [`PeerAuthentication`](https://istio.io/latest/docs/reference/config/security/peer_authentication/)
    - [`Gateway`](https://istio.io/latest/docs/reference/config/networking/gateway/)
    - [`AuthorizationPolicy`](https://istio.io/latest/docs/reference/config/security/authorization-policy/)
    - [`RequestAuthentication`](https://istio.io/latest/docs/reference/config/security/request_authentication/)
4. Use `istioctl analyze -A` to check for potential issues in the Istio configuration. This command checks for potential issues in the Istio configuration and provides suggestions on how to fix them.
5. Check response flags: DC (Downstream client terminated connection), UC (Upstream terminated connection) are out of scope for Istio team, since it relates to client(DC) or workload application(UC) behavior. You can find them in the Envoy sidecar's access logs.

## Sidecar injection

1. Inspect if Istio CR is in Warning state, if yes check for the warning message. It might be helpful at the beginning of the investigation.
2. Check if the label `istio-injection=enabled` is present on the namespace where the application is deployed. 
3. Sidecar injection behavior set on namespace can be overwritten by the `sidecar.istio.io/inject` label on the pod. Make sure that this label is not set to `false` there.
4. Make sure the pod does not have `hostNetwork: true` in the spec. If it does, the sidecar will not be injected.

## Still something does not work?
1. Check the [Official Istio Troubleshooting Guide](https://github.com/istio/istio/wiki/Troubleshooting-Istio).
2. Look for already existing issues in our [Istio Module](https://github.com/kyma-project/istio/issues) repository. If none of them is related to your problem, create a new issue.