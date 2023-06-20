## Istio Controller

### Overview

Istio Controller is part of Istio Operator. Its role is to manage the installation of Istio as defined by the Istio custom resource (CR). The controller is responsible for tasks such as installing, updating, and uninstalling Istio.

### Istio version

The version of Istio is coupled with the version of the controller. That means that the Istio upgrade is triggered when you deploy a new version of the controller.

### Istio CR

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio CR that is used to manage the Istio installation. You can find a sample CR [here](config/samples/operator_v1alpha1_istio.yaml).  
Applying this CR triggers the installation of Istio, and deleting it triggers the uninstallation of Istio.

### Restart of workloads with enabled sidecar injection

When the Istio version is upgraded or the resource configuration of the proxies changes, the Pods with enabled Istio injection are restarted.
It's done automatically for all resources that allow the rolling restart.
If a resource is a job, a ReplicaSet that is not managed by any deployment, or a Pod that is not managed by any other resource, the restart cannot be performed automatically. In these cases, a warning is logged, and you must manually restart the resources.

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
