## Istio Controller

### Overview

Istio Controller is part of Istio Operator. Its role is to manage the installation of Istio as defined by the Istio custom resource (CR). The controller is responsible for tasks such as installing, updating, and uninstalling Istio.

### Istio version

The version of Istio is dependent on the version of Istio Controller that you use. This means that if a new version of Istio Controller introduces a new version of Istio, deploying the controller will automatically trigger an upgrade of Istio.

### Istio CR

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio CR that is used to manage the Istio installation. To learn more, read the [Istio CR documentation](./01-20-istio-custom-resource).

### Restart of workloads with enabled sidecar injection

When Istio version is updated or the configuration of the proxies is changed, the Pods that have Istio injection enabled are automatically restarted. This is possible for all resources that allow for a rolling restart. If Istio is uninstalled, the workloads are restarted again to remove the sidecars.
However, if a resource is a job, a ReplicaSet that is not managed by any deployment, or a Pod that is not managed by any other resource, the restart cannot be performed automatically. In such cases, a warning is logged, and you must manually restart the resources.

### Supported use cases

- Install, upgrade, and uninstall Istio.
- Restart workloads that have a proxy sidecar to ensure that these workloads are using the correct Istio version.

### Status codes

|   Code         | Description                                  |
|:--------------:|:---------------------------------------------|
|  `Ready`       | Controller finished reconciliation.          |
|  `Processing`  | Controller is installing or upgrading Istio. |
|  `Deleting`    | Controller is uninstalling Istio.            |
|  `Error`       | An error occurred during reconciliation.     |
