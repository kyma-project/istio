# Technical Design

## Istio Operator
- Istio version is defined through IstioCR
  - We support three versions of Istio at a time. A validation webhook will make sure that IstioCR the applied istio version is valid.
- Grafana dashboards and resources are not handled by the operator.

### Installation & Upgrade of Istio
Istio installation, upgrade and uninstall is done using [Istio Go client module](https://github.com/istio/client-go).  
The Upgrade process won't support canary upgrades in the first version of the operator.

### Reconciliation
The reconciliation is done by multiple reconcilers. Each reconciler must have its clearly separated responsibility and should be self-contained.
This means the reconciler should work in isolation when assessing whether reconciliation is required, applying changes and returning a status.
Although we want the reconcilers to be as independent as possible, there can be dependencies between them (mainly on the `IstioInstallationReconciler`).
The goal is to keep the reconcilers as decoupled and independent as possible.

#### Components
##### IstioReconciler
This is the overall reconciler that takes care of the entire istio reconciliation process. 
The responsibility of this reconciler is to create the fetch the current configuration and pass it on to the other reconciler together with the desired configuration. 
It also controls the overall reconciliation process by running the reconcilers considering the dependencies between them.

##### IstioInstallationReconciler
This reconciler handles the installation, upgrade and uninstallion of Istio in the cluster. The reconciler als creates the IstioOperator
that will be used in combination with the [Istio Go client module](https://github.com/istio/client-go). 
This IstioOperator is created by merging the IstioCR with the IstioOperator with Kyma default values. 

##### ProxySidecarReconciler
This reconciler depends on the `IstioInstallationReconciler`. Its responsibility is to restart pods based on specific configuration changes.
As of now the following scenarios must be covered by this reconciler:
- Restart pods with proxy sidecar when CNI config changed
- Restart pods with proxy sidecar after Istio version update
- Restart pods without proxy sidecar, because of Istio downtime

##### IstioIngressGatewayReconciler
This reconciler depends on the `IstioInstallationReconciler`. Its responsibility is to restart the Istio ingress gateway ingress gate  based on specific configuration changes.
As of now the following scenarios must be covered by this reconciler:
- Restart when `numTrustedProxies` changed.

##### MonitoringReconciler
This reconciler depends on the `IstioInstallationReconciler` and it applies resources for monitoring the istio installation.
As of now the following resources are part of the monitoring:
- `ServiceMonitor` resource for Istio used by the monitoring module 
- `VirtualService` for monitoring of the Istio health by an external system

#### TODO PeerAuthenticationReconciler


### Scenario user bring their own Istio
In this scenario API gateway would support defined Istio versions. A user can then install one of the supported istio versions.
There should be a documentation to explain what needs to be configured to expose a ServiceMonitor for the monitoring module.