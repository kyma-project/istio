# Istio Managed ClusterRoles

Learn about the ClusterRoles managed by the Istio module and their permissions.

## Overview

To further streamline the management of permissions for the Istio module,
we have introduced new ClusterRoles that support Kubernetes native role aggregation.
These ClusterRoles are designed to provide a more modular and flexible approach
to managing permissions for the Istio module and its resources.

## Roles

By default, when you install the Istio module, it creates the following ClusterRoles:

- `kyma-istio-view` - Grants read-only access to Istio module resources (the `Istio` custom resource).
- `kyma-istio-edit` - Grants read and write access to Istio module resources (the `Istio` custom resource).
- `kyma-istio-resources-view` - Grants read-only access to all resources from all API groups handled by Istio.
- `kyma-istio-resources-edit` - Grants read and write access to all resources from all API groups handled by Istio.

## Aggregation

All ClusterRoles implement native Kubernetes role aggregation.
This functionality is handled by the Kubernetes controller manager.
If you have a binding for any of the default roles (`view` and `edit`),
you automatically get the permissions defined in these ClusterRoles without
needing to create additional bindings.

The mapping of the default roles to the Istio module managed ClusterRoles is as follows:

| General-purpose Role | Istio module managed Cluster Role |
|----------------------|-----------------------------------|
| `viewer`             | `kyma-istio-view`                 |
| `viewer`             | `kyma-istio-resoures-view`        |
| `edit`               | `kyma-istio-edit`                 |
| `edit`               | `kyma-istio-resoures-edit`        |

## Validation

To check what ClusterRoles were created with the Istio module, run the following command:

```sh
kubectl get clusterroles -l "kyma-project.io/module=istio"
```
