# Gateway API CRDs Management

The Istio module installs and manages the standard [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) CustomResourceDefinitions (CRDs) as part of its reconciliation loop.

## Overview

The Istio module installs the following Gateway API CRDs on the cluster:

| CRD Name                                       | API Group                            | Kind               |
|------------------------------------------------|--------------------------------------|--------------------|
| `backendtlspolicies.gateway.networking.k8s.io` | `gateway.networking.k8s.io/v1alpha3` | `BackendTLSPolicy` |
| `gatewayclasses.gateway.networking.k8s.io`     | `gateway.networking.k8s.io/v1`       | `GatewayClass`     |
| `gateways.gateway.networking.k8s.io`           | `gateway.networking.k8s.io/v1`       | `Gateway`          |
| `grpcroutes.gateway.networking.k8s.io`         | `gateway.networking.k8s.io/v1`       | `GRPCRoute`        |
| `httproutes.gateway.networking.k8s.io`         | `gateway.networking.k8s.io/v1`       | `HTTPRoute`        |
| `listenersets.gateway.networking.k8s.io`       | `gateway.networking.k8s.io/v1alpha2` | `ListenerSet`      |
| `referencegrants.gateway.networking.k8s.io`    | `gateway.networking.k8s.io/v1beta1`  | `ReferenceGrant`   |

All module-managed CRDs are labeled with `kyma-project.io/module: istio`.

The primary CRD (`gateways.gateway.networking.k8s.io`) additionally carries the label `kyma-project.io/managed-gateway-api: "true"`, which can be used to detect whether Gateway API is managed by the Istio module.

## Behavior

### Installation

When the Istio module is installed or updated, the controller checks the primary CRD (`gateways.gateway.networking.k8s.io`) to decide how to proceed:

- **Primary CRD does not exist**: The controller creates all Gateway API CRDs and labels each with `kyma-project.io/module: istio`. The primary CRD also receives `kyma-project.io/managed-gateway-api: "true"`.
- **Primary CRD exists and is module-managed** (has the `kyma-project.io/managed-gateway-api: "true"` label): The controller updates all CRDs to the bundled version.
- **Primary CRD exists but is not module-managed** (missing the `kyma-project.io/managed-gateway-api: "true"` label): The controller leaves all CRDs unchanged and sets the Istio CR status to `Warning` with the reason `GatewayAPICRDsAlreadyInstalled`.

### Uninstallation

When the Istio CR is deleted:

- Module-managed Gateway API CRDs are deleted together with the rest of the Istio resources.
- CRDs without the `kyma-project.io/module: istio` label are left on the cluster unchanged.
- If any user-created Gateway API resources (for example, `HTTPRoute`, `Gateway`) are present on the cluster **and** the primary module-managed Gateway API CRD exists, the deletion is blocked. The Istio CR status is set to `Warning` with the reason `GatewayAPIResourcesDangling` until all blocking resources are removed.

## Pre-existing Gateway API CRDs

If your cluster already has Gateway API CRDs installed by another tool (for example, from an upstream release), the Istio module does not overwrite them. The Istio CR transitions to the `Warning` state with the following condition:

| Field     | Value                                                                                                                                                                                     |
|-----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `type`    | `Ready`                                                                                                                                                                                   |
| `status`  | `False`                                                                                                                                                                                   |
| `reason`  | `GatewayAPICRDsAlreadyInstalled`                                                                                                                                                          |
| `message` | `Gateway API CRDs are already installed. To allow Kyma Istio module to manage them, add the label kyma-project.io/managed-gateway-api=true to the gateways.gateway.networking.k8s.io CRD` |

To verify the condition, run:

```bash
kubectl get istio default -n kyma-system -o jsonpath='{.status.conditions}'
```

### Allowing the Module to Manage Pre-existing CRDs

To let the Istio module take over management of already installed Gateway API CRDs, add the `managed-gateway-api` label to the primary CRD:

```bash
kubectl label crd gateways.gateway.networking.k8s.io kyma-project.io/managed-gateway-api=true
```

After labeling the primary CRD, the Istio module manages all Gateway API CRDs going forward, including updates during module upgrades and deletion when the module is removed. The module also labels the remaining CRDs with `kyma-project.io/module: istio` during the next reconciliation.

## Blocking Deletion Due to Gateway API Resources

If you try to delete the Istio module while user-created Gateway API resources (such as `HTTPRoute` or `Gateway` objects) still exist on the cluster **and** the primary module-managed Gateway API CRD is present, the deletion is blocked. The Istio CR transitions to the `Warning` state with the following condition:

| Field     | Value                                                                    |
|-----------|--------------------------------------------------------------------------|
| `type`    | `Ready`                                                                  |
| `status`  | `False`                                                                  |
| `reason`  | `GatewayAPIResourcesDangling`                                            |
| `message` | `Gateway API deletion blocked because of existing Gateway API resources` |

To identify which resources are blocking deletion, inspect the `istio-controller-manager` logs:

```bash
kubectl logs -n kyma-system -l app=istio-controller-manager --tail=100 | grep "Gateway API resource is blocking"
```

Once you remove those resources, the Istio module deletion proceeds automatically.
