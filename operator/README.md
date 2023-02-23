# Istio Controller

## Overview
The Istio controller is part of the Istio Manager and manages an Istio installation defined by the Istio custom resource.
The controller takes care of installing, updating and uninstalling Istio.

## Istio Version
The version of Istio is coupled with the version of the controller. That means an Istio upgrade is triggered by deploying a
new version of the controller.

## Istio Custom Resource
The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) is the description of Istio custom resource that is used
to manage the istio installation. A sample resource can be found [here](config/samples/operator_v1alpha1_istio.yaml).  
Applying this custom resource triggers the installation of Istio and deleting it triggers an uninstallation of Istio.

## Supported Use Cases
- Install, upgrade and uninstall Istio
- Restarts workloads that have a proxy sidecar to ensure that these workloads are using the correct version

## Status codes

|      Code      | Description                                 |
|:--------------:|:--------------------------------------------|
|   **Ready**    | Controller finished reconciliation          |
| **Processing** | Controller is installing or upgrading Istio |
|  **Deleting**  | Controller is uninstalling Istio            |
|   **Error**    | An error happened during reconciliation     |


