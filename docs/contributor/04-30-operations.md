# Managing Istio with Istio Operator

This document contains detailed descriptions of the operations that Istio Operator is responsible for.

- [Managing Istio with Istio Operator](#managing-istio-with-istio-operator)
  - [Installation of Istio](#installation-of-istio)
    - [Istio upgrade version checking](#istio-upgrade-version-checking)
  - [Reconciliation of Istio](#reconciliation-of-istio)
    - [Reconciliation interval](#reconciliation-interval)
  - [Uninstallation of Istio](#uninstallation-of-istio)
  - [Scenario: Users bring their own Istio installation](#scenario-users-bring-their-own-istio-installation)

## Installation of Istio

The Istio installation, upgrade and uninstallation are performed using the [Istio Go module](https://github.com/istio/istio).

In the sample implementation, the [istio.Install function](https://github.tools.sap/xf-goat/kyma-istio-operator/blob/ec0f99786408407b4a6d8b79abe3af6c389cd35d/controllers/servicemesh_controller.go#L73) is used for installation.
The installation of Istio is executed as a synchronous and blocking call that checks the proper status of the installation. This means that the reconciliation loop is blocked until Istio is installed.  
The installation or upgrade scenario is not often executed. The call to the `Install` function should be protected by checks so that it is only executed when necessary.
Therefore, we have agreed that it is okay to block the reconciliation loop during the installation of Istio.

The following diagram shows the reconciliation process for installing, uninstalling, and canary upgrading (using revisions) Istio.

![Istio Installation Reconciliation](/docs/assets/istio-installation-reconciliation.svg)

### Istio upgrade version checking

You can upgrade Istio only by one minor version (for example, 1.2.3 -> 1.3.0). Reconciliation fails if the difference between current and target minor versions is greater than one (for exmple, 1.2.3 -> 1.4.0).
An upgrade of a major version fails (for example, 1.2.3 -> 2.0.0), as well as any downgrade (for example, 1.2.3 -> 1.2.2).

## Reconciliation of Istio

The reconciliation loop of Istio is based on the [Istio CR](https://github.com/kyma-project/istio/blob/main/docs/xff-proposal.md) custom resource and is controlled by `IstioController`. This controller contains several self-contained components, which we have suffixed with reconciliation.
We decided to split the logic in these reconciliation components to have a better extensibility and maintainability. This means each of these components must have its clearly separated responsibility
and must work in isolation when assessing whether reconciliation is required, applying changes, and returning a status.
The execution of the reconciliation must be fast, and we must avoid many blocking calls. Long-running tasks must be executed asynchronously, and the status must be evaluated in the next reconciliation cycle.

 The following diagram shows the reconciliation loop of `IstioController`:
![Reconciliation Loop Diagram](/docs/assets/istio-controller-reconciliation-loop.svg)

### Reconciliation interval

Since the Isito module deals with security-related topics, we want to perform the reconciliation as often as possible.
Not only do we want to reconcile when [Istio CR](https://github.com/kyma-project/istio/blob/main/docs/xff-proposal.md) changes, but also to verify regularly if resources remain unchanged and are in the expected state.  
The reconciliation frequency of a manager is determined by the [SyncPeriod](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Options). By default, it is set to 10 hours.
To match the desired reconciliation interval, use one of the following options:

- Change the `SyncPeriod`, so it matches the desired value.
- Always return `RequeueAfter` in the result of the Reconcile function to trigger the next reconciliation:

```go
func Reconcile(ctx context.Context, o reconcile.Request) (reconcile.Result, error) {
 // Implement business logic of reading and writing objects here
 return reconcile.Result{RequeueAfter: : 5 * time.Minute}, nil
}
```

The time needed to perform the reconciliation can vary a lot, so choosing an appropriate interval might be challenging. Small changes may only require
restarting the sidecar proxies or the Ingress gateway. Therefore, they are much faster than a new installation or a Canary upgrade.

We can start with using `SyncPeriod` set to 5 minutes. For now, we only want to have a single controller, so it is not a problem to start with a higher time-based reconciliation. When we add more controllers we can use `RequeueAfter`
in order to trigger them at different intervals, or to have a different interval while a long-running process like installation is executed.

The queuing of reconciliation requests is handled by [controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime) and is out of scope of this design.

## Uninstallation of Istio

The default behaviour triggered on deletion of Istio Custom Resource is to uninstall all of Istio components only if there are none customer created resources present on the cluster. This behaviour is called `blocking` deletion strategy and will take place unless the intent to delete all resources, including non default Istio ones, is explicitly defined by selecting `cascading` deletion strategy.

> TODO: At this moment only `blocking` strategy is implemented and triggered by default. Implement `cascading` strategy as described in this [issue](https://github.com/kyma-project/istio/issues/130).

## Scenario: Users bring their own Istio installation

In this scenario, API Gateway supports the defined Istio versions. The user can then install one of the supported Istio versions.
There should be documentation explaining what needs to be configured to expose a ServiceMonitor for the monitoring module.