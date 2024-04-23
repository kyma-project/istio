# Istio Custom Resource Status Improvements

## Status
Accepted

## Context
The Istio status doesn't contain enough information to easily debug without directly accessing Kyma Istio Operator logs. We would like to make it easier both for the user and the Kyma Istio team to view the state of the Istio module without that necessity.

## Decision

For the sake of simplicity, we implement the condition of type `Ready` that supports multiple reasons. This way, we can reduce the number of condition types. If we use the [SetStatusCondition function](https://pkg.go.dev/k8s.io/apimachinery/pkg/api/meta#SetStatusCondition) of [k8s.io/apimachinery](https://github.com/kubernetes/apimachinery), the **lastTransitionTime** is only updated when the status of the condition changes.

To support multiple reasons for the status `False`, we decided to set the status of the `Ready` condition to `Unknown` at the beginning of the reconciliation. This allows us to force the **lastTransitionTime** of the `Ready` condition to always be updated.

In addition, the `Ready` condition reflects one reconciliation status. Therefore, it is a good idea to set it to `Unknown` at the beginning.

We also have conditions that reflect conditional behavior. In the first implementation, we have the condition of type `ProxySidecarRestartSucceeded`. For such conditions, we don't want to reset the status to `Unknown` on every reconciliation but only update the status when the related action (in this case, the proxy restart) is executed. This provides clearer feedback in the CR about when a specific transition occurred.

Conditions:

| CR state   | Type                         | Status | Reason                            | Message                                                                                  |
|------------|------------------------------|--------|-----------------------------------|------------------------------------------------------------------------------------------|
| Ready      | Ready                        | True   | ReconcileSucceeded                | Reconciliation succeeded                                                                 |
| Error      | Ready                        | False  | ReconcileFailed                   | Reconciliation failed                                                                    |
| Error      | Ready                        | False  | OlderCRExists                     | This Istio custom resource is not the oldest one and does not represent the module state |
| Processing | Ready                        | False  | IstioInstallNotNeeded             | Istio installation is not needed                                                         |
| Processing | Ready                        | False  | IstioInstallSucceeded             | Istio installation succeeded                                                             |
| Processing | Ready                        | False  | IstioUninstallSucceeded           | Istio uninstallation succeeded                                                            |
| Error      | Ready                        | False  | IstioInstallUninstallFailed       | Istio install or uninstall failed                                                        |
| Error      | Ready                        | False  | IstioCustomResourceMisconfigured  | Istio custom resource has invalid configuration                                          |
| Warning    | Ready                        | False  | IstioCustomResourcesDangling      | Istio deletion blocked because of existing Istio custom resources                        |
| Processing | Ready                        | False  | CustomResourcesReconcileSucceeded | Custom resources reconciliation succeeded                                                |
| Error      | Ready                        | False  | CustomResourcesReconcileFailed    | Custom resources reconciliation failed                                                   |
| Processing | ProxySidecarRestartSucceeded | True   | ProxySidecarRestartSucceeded      | Proxy sidecar restart succeeded                                                          |
| Error      | ProxySidecarRestartSucceeded | False  | ProxySidecarRestartFailed         | Proxy sidecar restart failed                                                             |
| Warning    | ProxySidecarRestartSucceeded | False  | ProxySidecarManualRestartRequired | Proxy sidecar manual restart is required for some workloads                              |
| Processing | Ready                        | False  | IngressGatewayReconcileSucceeded  | Istio Ingress Gateway reconciliation succeeded                                           |
| Error      | Ready                        | False  | IngressGatewayReconcileFailed     | Istio Ingress Gateway reconciliation failed                                              |


## Consequences
This architectural decision allows our team and customers to monitor the state of the Istio module on the cluster more easily. Previously, accessing the Istio module logs was often required to gain better visibility into the issues occurring in the cluster.

Since it becomes a part of our API, we must ensure the stability.