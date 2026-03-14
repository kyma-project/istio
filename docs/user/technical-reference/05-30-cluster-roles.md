# Istio managed Cluster Roles

Get familiar with the Cluster Roles that are managed by the Istio module and their permissions.

## Overview

To further streamline the management of permissions for the Istio module, we've introduced new Cluster Roles that support Kubernetes native role aggregation. 
These Cluster Roles are designed to provide a more modular and flexible approach to managing permissions for the Istio module and its resources.

## Roles

By default, when you install the Istio module, it creates the following Cluster Roles:
- `kyma-istio-view` - grants read-only access to Istio module resources (`Istio` Custom Resource).
- `kyma-istio-edit` - grants read and write access to Istio module resources (`Istio` Custom Resource).

Additionally, when `Istio` Custom Resource is created, the following Cluster Roles are generated and managed by the module:
- `kyma-istio-resoures-view` - grants read-only access to all resources from all API groups handled by Istio.
- `kyma-istio-resoures-edit` - grants read and write access to all resources from all API groups handled by Istio.

## Aggregation

All Cluster Roles implement native Kubernetes role aggregation. This functionality is handled by Kubernetes controller manager. 
If the user has a binding for any of the general-purpose roles (`viewer` and `edit`), they will automatically get
permissions defined in the above Cluster Roles without needing to create additional bindings for them.

The mapping of the general-purpose roles to the specific Cluster Roles is as follows:

| General-purpose Role | Istio module managed Cluster Role |
|----------------------|-----------------------------------|
| `viewer`             | `kyma-istio-view`                 |
| `viewer`             | `kyma-istio-resoures-view`        |
| `edit`               | `kyma-istio-edit`                 |
| `edit`               | `kyma-istio-resoures-edit`        |

## Validation

To check what Cluster Roles have been created by Istio module, you can run the following commands:

```sh
kubectl get clusterroles -l "kyma-project.io/module=istio"