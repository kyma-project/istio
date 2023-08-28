# Istio Controller

## Overview

Istio Controller is part of Kyma Istio Operator. Its role is to manage the installation of Istio as defined by the Istio custom resource (CR). The controller is responsible for:
- Installing, upgrading, and uninstalling Istio
- Restarting workloads that have a proxy sidecar to ensure that these workloads are using the correct Istio version.

## Istio version

The version of Istio is dependent on the version of Istio Controller that you use. This means that if a new version of Istio Controller introduces a new version of Istio, deploying the controller will automatically trigger an upgrade of Istio.

## Istio CR

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio CR that is used to manage the Istio installation. To learn more, read the [Istio CR documentation](../03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md).

## Restart of workloads with enabled sidecar injection

When the Istio version is updated or the configuration of the proxies is changed, the Pods that have Istio injection enabled are automatically restarted. This is possible for all resources that allow for a rolling restart. If Istio is uninstalled, the workloads are restarted again to remove the sidecars.
However, if a resource is a job, a ReplicaSet that is not managed by any deployment, or a Pod that is not managed by any other resource, the restart cannot be performed automatically. In such cases, a warning is logged, and you must manually restart the resources.

## Status codes

|     Code     | Description                                  |
|:------------:|:---------------------------------------------|
|   `Ready`    | Controller finished reconciliation.          |
| `Processing` | Controller is installing or upgrading Istio. |
|  `Deleting`  | Controller is uninstalling Istio.            |
|   `Error`    | An error occurred during reconciliation.     |
|  `Warning`   | Controller is misconfigured.                 |
