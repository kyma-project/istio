# Technical Design

## Istio Operator

### Assumptions
- Istio version is defined through IstioCR
  - We support three versions of Istio at a time. A validation webhook will make sure that IstioCR the applied istio version is valid.
- Grafana dashboards and resources are not handled by the operator.
- The Istio upgrade process won't support canary upgrades in the first version of the operator.

### Installation & Upgrade of Istio
The Istio installation, upgrade and uninstall is done using [Istio Go client module](https://github.com/istio/client-go).

### Reconciliation
The reconciliation is done by multiple self-contained reconcilers to have a better extensibility and maintainability. This means each reconciler must have its clearly separated responsibility.
and must work in isolation when assessing whether reconciliation is required, applying changes and returning a status.  

Although we want the reconcilers to be as decoupled and independent as possible, there is an execution dependency as we first need to install/upgrade Istio ( done by `IstioInstallationReconciler`)
before the other reconcilers can be executed.

### Components
#### IstioReconciler
This is the reconciler that takes care of the entire istio reconciliation process. 
The responsibility of this reconciler is to create the fetch the current configuration and pass it on to the other reconciler together with the desired configuration. 
It also controls the reconciliation process by running the reconcilers considering the execution dependencies between them.

#### IstioInstallationReconciler
This reconciler handles the installation, upgrade and uninstall of Istio in the cluster. The reconciler also creates the IstioOperator
that will be used to apply changes to the Istio installation in combination with the [Istio Go client module](https://github.com/istio/client-go).
The IstioOperator is created by merging the `IstioCR` with the IstioOperator with Kyma default values. 

##### IstioManager
This component contains the logic for managing the Istio installation. It also knows the supported client versions and forwards the 
Istio API requests (e.g. Install, Upgrade) to the correct version of `IstioClient`.

##### IstioClient
The IstioClient encapsulates the [Istio Go client module](https://github.com/istio/client-go) for one specific Istio version. 
As we do not expect breaking changes in all new API versions, there should always be a default version that is used.

#### ProxySidecarReconciler
This reconciler must be executed after the `IstioInstallationReconciler`. Its responsibility is to restart pods based on specific configuration changes.

As of now the following scenarios must be covered by this reconciler:
- Restart pods with proxy sidecar when CNI config changed
- Restart pods with proxy sidecar after Istio version update
- Restart pods without proxy sidecar, because of Istio downtime

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