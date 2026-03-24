# Istio Managed ClusterRoles

Learn about the ClusterRoles that the Istio module manages and the permissions they grant.

## Overview

To further streamline the management of permissions for the Istio module,
we have introduced new ClusterRoles that support Kubernetes native role aggregation.
ClusterRoles streamline permission management and support Kubernetes native role aggregation.
They provide a modular and flexible approach to managing permissions for Istio resources.

## Roles

By default, when you install the Istio module, it creates the following ClusterRoles:

- `kyma-istio-view` - Grants read-only access to the Istio custom resource (CR).
- `kyma-istio-edit` - Grants read and write access to the Istio CR.
- `kyma-istio-resources-view` - Grants read-only access to all resources from all API groups handled by Istio.
- `kyma-istio-resources-edit` - Grants read and write access to all resources from all API groups handled by Istio.

## Aggregation

All ClusterRoles implement native Kubernetes role aggregation.
This functionality is handled by the Kubernetes controller manager.
When you have a binding for any of the general-purpose roles (`viewer` or `edit`), you automatically get the permissions
from the corresponding Istio-managed ClusterRoles.
You don't need to create separate role bindings for Istio resources.

The following table shows how general-purpose roles map to Istio module ClusterRoles:

| General-purpose Role | Istio module managed ClusterRole |
|----------------------|-----------------------------------|
| `viewer`             | `kyma-istio-view`                 |
| `viewer`             | `kyma-istio-resoures-view`        |
| `edit`               | `kyma-istio-edit`                 |
| `edit`               | `kyma-istio-resoures-edit`        |

## Validation

To check what ClusterRoles have been created by the Istio module, run the following command:

```sh
kubectl get clusterroles -l "kyma-project.io/module=istio"
```
