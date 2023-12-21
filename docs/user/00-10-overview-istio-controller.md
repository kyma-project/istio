# Istio Controller

## Overview

Istio Controller is part of Kyma Istio Operator. Its role is to manage the installation of Istio as defined by the Istio custom resource (CR). The controller is responsible for:
- Installing, upgrading, and uninstalling Istio
- Restarting workloads that have a proxy sidecar to ensure that these workloads are using the correct Istio version.

## Istio Version

The version of Istio is dependent on the version of Istio Controller that you use. This means that if a new version of Istio Controller introduces a new version of Istio, deploying the controller will automatically trigger an upgrade of Istio.

## Istio Custom Resource

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio CR that is used to manage the Istio installation. To learn more, read the [Istio CR documentation](04-00-istio-custom-resource.md).

## Restart of Workloads with Enabled Sidecar Injection

When the Istio version is updated or the configuration of the proxies is changed, the Pods that have Istio injection enabled are automatically restarted. This is possible for all resources that allow for a rolling restart. If Istio is uninstalled, the workloads are restarted again to remove the sidecars.
However, if a resource is a job, a ReplicaSet that is not managed by any deployment, or a Pod that is not managed by any other resource, the restart cannot be performed automatically. In such cases, a warning is logged, and you must manually restart the resources.

## Status Codes

|     Code     | Description                                  |
|:------------:|:---------------------------------------------|
|   `Ready`    | Controller finished reconciliation.          |
| `Processing` | Controller is installing or upgrading Istio. |
|  `Deleting`  | Controller is uninstalling Istio.            |
|   `Error`    | An error occurred during reconciliation.     |
|  `Warning`   | Controller is misconfigured.                 |

Conditions:

| CR state   | Condition type | Condition status | Condition reason             | Remark                                                                          |
|------------|----------------|------------------|------------------------------|---------------------------------------------------------------------------------|
| Ready      | Ready          | true             | ReconcileSucceeded           | Reconciliation succeeded                                                        |
| Ready      | Ready          | true             | UpdateCheckSucceeded         | Update not required                                                             |
| Ready      | Ready          | true             | UpdateDone                   | Update done                                                                     |
| Processing | Ready          | false            | Processing                   | Istio installation is proceeding                                                |
| Processing | Ready          | false            | UpdateCheck                  | Checking if update is required                                                  |
| Warning    | Ready          | false            | IstioCustomResourcesDangling | Istio deletion blocked because of existing Istio resources that are not default |
| Warning    | Ready          | false            | CustomResourceMisconfigured  | Configuration present on Istio Custom Resource is not correct                   |
| Deleting   | Ready          | false            | Deleting                     | Proceeding with uninstallation and deletion of Istio                            |
| Error      | Ready          | false            | IstioInstallationFailed      | Failure during execution of Istio installation                                  |
| Error      | Ready          | false            | OlderCRExists                | This CR is not the oldest one so does not represent the module State            |

## X-Forwarded-For HTTP Header

The **X-Forwarded-For** (XFF) header is only supported on AWS clusters.