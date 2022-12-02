# Technical Design

## To be discussed


## IstioOperator

![IstioOperator Overview](./istio-operator-overview.svg)

### Ownership of current resources in Kyma repository

When moving to a more modularised architecture, the [IstioOperator resources](https://github.com/kyma-project/kyma/tree/main/resources/istio), 
the [additional istio-resources](https://github.com/kyma-project/kyma/tree/main/resources/istio-resources) and 
the [certificates](https://github.com/kyma-project/kyma/tree/main/resources/certificates) must have clear ownership.


#### Assumptions 
- Grafana dashboards and resources are not owned by the operator.

### Handling of Istio version
We don't want to expose the version. That means that the version of Istio is coupled to the version of the operator. The benefit of this is,
that we are in full control of the versioning and hide this complexity from the user.  
That means if we want to release a new Istio version, we have to release a new version of the operator.

## IstioController

![Component Diagram](./controller-component-diagram.svg)

### Installation & Upgrade of Istio
The Istio installation, upgrade and uninstall is done using [Istio Go module](https://github.com/istio/istio).

### Reconciliation
The reconciliation is done by multiple self-contained reconcilers to have a better extensibility and maintainability. This means each reconciler must have its clearly separated responsibility
and must work in isolation when assessing whether reconciliation is required, applying changes and returning a status.  

Although we want the reconcilers to be as decoupled and independent as possible, there is an execution dependency as we first need to install/upgrade Istio ( done by `IstioInstallationReconciler`)
before the other reconcilers can be executed.

### Components
#### IstioReconciler
This is the reconciler that takes care of the entire Istio reconciliation process. 
The responsibility of this reconciler is to fetch the current configuration and pass it on to the other reconciler together with the desired configuration. 
It also controls the reconciliation process by running the reconcilers considering the execution dependencies between them.

#### IstioInstallationReconciler
This reconciler decides if an installation, upgrade or uninstall of Istio in the cluster must be done. The reconciler also creates the IstioOperator
which is used to make changes to the Istio installation by passing it to the `IstioManager`.

The installed IstioOperator is created by merging the `IstioCR` with the IstioOperator with Kyma default values.

##### IstioManager
This component contains the logic for managing the Istio installation. It knows about the supported client versions and forwards the 
Istio API requests (e.g. Install, Upgrade) to the correct version of `IstioClient` if there are breaking changes.

##### IstioClient
A IstioClient encapsulates a specific version of the [Istio Go module](https://github.com/istio/istio). 
As we want to support canary updates at some point we might need to support two version of the library if there are breaking changes.

#### ProxySidecarReconciler
This reconciler must be executed after the `IstioInstallationReconciler`. Its responsibility is to restart pods based on specific configuration changes.

As of now the following scenarios must be covered by this reconciler:
- Restart pods with proxy sidecar when CNI config changed
- Restart pods with proxy sidecar after Istio version update
- Restart pods without proxy sidecar, because of Istio downtime
- Restart pods with proxy sidecar when proxy resources change

#### IstioIngressGatewayReconciler
This reconciler must be executed after the `IstioInstallationReconciler`. Its responsibility is to restart the Istio ingress gateway ingress gate  based on specific configuration changes.

As of now the following scenarios must be covered by this reconciler:
- Restart when `numTrustedProxies` changed.

#### MonitoringReconciler
This reconciler must be executed after the `IstioInstallationReconciler` and it applies resources for monitoring the istio installation.

As of now the following resources are part of the monitoring:
- `ServiceMonitor` resource for Istio used by the monitoring module 
- `VirtualService` for monitoring of the Istio health by an external system

#### PeerAuthenticationReconciler
This reconciler must be executed after the `IstioInstallationReconciler` and it applies a PeerAuthentication that configures
the default mTLS mode in the cluster.


## Scenario: Users bring their own Istio installation
In this scenario the API Gateway would support defined Istio versions. A user can then install one of the supported istio versions.
There should be a documentation to explain what needs to be configured to expose a ServiceMonitor for the monitoring module.