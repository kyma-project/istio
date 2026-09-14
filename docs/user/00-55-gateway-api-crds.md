# Gateway API CRDs Management

The Istio module installs and manages [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) CRDs in your cluster. If Gateway API CRDs are already installed by another tool, you can hand over their management to the Istio module.

## Installing Gateway API CRDs

If no Gateway API CRDs exist in your cluster when you add the Istio module, the module installs them automatically. You can immediately start using Gateway API resources without any additional steps.

For the full list of CRDs installed by the Istio module, see the [Gateway API specification](https://gateway-api.sigs.k8s.io/reference/api-spec/main/spec/#gatewaynetworkingk8siov1).

The Istio module labels all managed CRDs with `kyma-project.io/module: istio`. The `gateways.gateway.networking.k8s.io` CRD also carries the `kyma-project.io/managed-gateway-api: "true"` label, which indicates that the Istio module is responsible for managing Gateway API in the cluster.     


## Pre-existing Gateway API CRDs

If Gateway API CRDs are already installed in your cluster by another tool (for example, from an upstream release), the Istio module does not overwrite them. The Istio CR transitions to the `Warning` state with the **Ready** condition set to `false` and the reason `GatewayAPICRDsAlreadyInstalled`.

To verify the condition, run:

```bash
kubectl get istio default -n kyma-system -o jsonpath='{.status.conditions[0]}'
```

## Hand Over Gateway API Management to the Istio Module

If you hand over management of pre-existing Gateway API CRDs to the Istio module, the CRDs are automatically kept up to date during module upgrades and removed cleanly when the module is deleted.

To do this, add the `kyma-project.io/managed-gateway-api=true` label to the `gateways` CRD:

```bash
kubectl label crd gateways.gateway.networking.k8s.io kyma-project.io/managed-gateway-api=true
```

## Deleting the Istio Module with Gateway API Resources

When you delete the Istio module, the module-managed Gateway API CRDs are removed together with the rest of the Istio resources. CRDs that were not installed by the Istio module are left on the cluster unchanged.

To protect your existing Gateway API resources from being orphaned, deletion is blocked if any resources (such as `HTTPRoute` or `Gateway` objects) still exist on the cluster. The Istio CR transitions to the `Warning` state with the **Ready** condition set to `false` and the reason `GatewayAPIResourcesDangling`. If deletion is blocked, see [Reverting the Istio Module's Deletion](./troubleshooting/03-50-recovering-from-unintentional-istio-removal.md).
