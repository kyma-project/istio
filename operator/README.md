# Istio controller

## Overview
The Istio controller is part of Istio-Manager and manages the Istio installation defined by the Istio custom resource.
The controller takes care of installing, updating, and uninstalling Istio.

## Istio version
The version of Istio is coupled with the version of the controller. That means that the Istio upgrade is triggered by deploying a
new version of the controller.

## Istio custom resource
The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the Istio custom resource that is used
to manage the Istio installation. You can find a sample custom resource [here](config/samples/operator_v1alpha1_istio.yaml).  
Applying this custom resource triggers the installation of Istio, and deleting it triggers the uninstallation of Istio.

## Supported use cases
- Install, upgrade, and uninstall Istio.
- Restart workloads that have a proxy sidecar to ensure that these workloads are using the correct Istio version.

## Status codes

|   Code         | Description                                  |
|:--------------:|:---------------------------------------------|
|  `Ready`       | Controller finished reconciliation.          |
|  `Processing`  | Controller is installing or upgrading Istio. |
|  `Deleting`    | Controller is uninstalling Istio.            |
|  `Error`       | An error occurred during reconciliation.     |


