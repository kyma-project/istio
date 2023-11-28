# Technical Design

## Kyma Istio Operator

The Kyma Istio Operator implements one controller that consists of several self-contained reconciliation components. The logic is split up in reconciliation components to 
have a better extensibility and maintainability. This means each of these components must have its clearly separated responsibility and must work in isolation when assessing whether reconciliation is required, applying changes, and returning a status.
To understand the reasons for the technical design of the Kyma Istio Operator, refer to the [Architecture Decision Record](https://github.com/kyma-project/istio/issues/135).

The following diagram illustrates the Kyma Istio Operator and its components:

![Kyma Istio Operator Overview](../assets/istio-operator-overview.svg)

## Istio CR

The [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md) a namespace-scoped resource that is used to manage the Istio installation. The reason for Istio CR being namespace-scoped is that it was created and used before the Kyma Istio Operator was introduced.
There is no advantage to the Istio CR being namespace-scoped as the Kyma Istio Operator only supports one Istio CR per cluster. However, it is not possible to change it to cluster-scoped without breaking changes.

If multiple Istio CRs exist on the cluster, the Kyma Istio operator will only use the oldest Istio CR for reconciliation and put the other Istio CRs in an error state.

## Istio version

The version of Istio is coupled to the version of the Kyma Istio Operator. This means that a particular version of the operator is responsible for a particular version of Istio and a new version of the operator is released to support a new version of Istio.
Upgrading to a new version of the operator will automatically update Istio if the version of Istio has changed in the new operator version.

## Version upgrade

The Istio version upgrade is limited to supporting only one minor version (1.2.3 -> 1.3.0). This implies that this is also relevant when upgrading the Kyma Istio Operator to a new version.
This means that when upgrading the operator to a new version, a minor version of the operator can only be skipped if the difference between the Istio versions in the new operator version is no more than one minor version.

If the difference between the current and the target version of Istio is greater than one minor version (1.2.3 -> 1.4.0), the reconciliation will fail.
The reconciliation will also fail for major version upgrades (1.2.3 -> 2.0.0) and version downgrades, as these are also not supported.

## Istio Controller

The Istio Controller is a [Kubernetes controller](https://kubernetes.io/docs/concepts/architecture/controller/), which is implemented using the [Kubebuilder](https://book.kubebuilder.io/) framework.
The controller is responsible for handling the [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md).

### Reconciliation
The [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md) is reconciled with each change to the **Spec** field. If no changes have been made, the reconciliation process occurs at the default interval of 10 hours.
You can use the [Istio Controller parameters ](../user/03-technical-reference/configuration-parameters/01-10-istio-controller-parameters.md) to adjust this interval.
If there is a failure during the reconciliation process, the default behavior of the [Kubernetes controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime) is to use exponential backoff requeue.

When [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md) is deleted, the default behavior is to uninstall all Istio components, but only if there are no customer-created resources on the cluster. This behavior is known as the blocking deletion strategy.

Before deleting the [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md), the default behavior is to uninstall all Istio components, but only if there are no customer-created Istio resources on the cluster. This behavior is known as the blocking deletion strategy. If any such resources are found, they are listed in the logs of the controller, and the [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md)'s status is set to `Warning` to indicate that there are resources blocking the deletion.
The `istios.operator.kyma-project.io/istio-installation` finalizer protects the deletion of the [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md). Once no more customer-created Istio resources that are blocking the deletion of the [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md), the Istio CR is deleted.

As part of the reconciliation loop, the controller invokes the reconciliation components.
The reconciliation loop is illustrated in the following diagram:

![Istio Controller Reconciliation Loop Diagram](../assets/istio-controller-reconciliation-loop.svg)

## Reconciliation components

Each reconciliation component must be completely independent and can calculate what to do during the reconciliation independently of the reconciliation of other components and based only on the state in a cluster and [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md).

### Istio InstallationReconciliation

This component handles the Istio installation. To be able to do this, it also creates the [IstioOperator CR](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/) which is used to apply changes to the Istio installation. 
The applied `IstioOperator CR` is created by merging `Istio CR` with IstioOperator containing Kyma default values.

#### Istio installation

The Istio installation, upgrade and uninstallation is performed using the [Istio Go module](https://github.com/istio/istio).
The installation of Istio is executed as a synchronous and blocking call that checks the proper status of the installation. This means that the reconciliation loop is blocked until Istio is installed.  
For each reconciliation the Istio installation is checked and if it is not installed, the installation is executed.
As part of the installation, this component adds the finalizer `istios.operator.kyma-project.io/istio-installation` to Istio CR, which is only removed after a successful uninstallation of Istio.

#### Istio CR lastAppliedConfiguration
The component adds the `operator.kyma-project.io/lastAppliedConfiguration` annotation to the [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md) and
updates it after each successful reconciliation. This annotation is used to compare the new state of the Istio CR with the previous state 
and to decide whether a certain configuration has been changed that results in a certain action, for example restarting the Ingress Gateway.

#### Istio Ingress Gateway restart
The component also detects changes in the `numTrustedProxies` configuration and restarts the Istio Ingress Gateway accordingly. 
Whenever a change in the `numTrustedProxies` configuration is detected, the Pods in the `istio-system/istio-ingressgateway` deployment are restarted.

### Istio ResourcesReconciliation

Istio ResourcesReconciliation is a component responsible for applying resources dependent on Istio, such as VirtualService or EnvoyFilter, and ensuring that the state of the Istio service mesh is configured correctly based on those resources.
To maintain the correct state, the component provides [Restart Predicates](#restart-predicates) on a per-resource basis, which are consumed by the [IngressGatewayReconciler](#ingressgatewayreconciler) and [ProxySidecarReconciliation](#proxysidecarreconciliation) components.
To understand the reasons for the predicates, refer to the [Architecture Decision Record](https://github.com/kyma-project/istio/issues/278).

For resources that are reconciled by this component, which are not Istio resources and would therefore remain on the cluster if Istio is uninstalled, the OwnerReference is set to the Istio CR. 
This means that if the Istio CR is deleted, these resources are also deleted.

### ProxySidecarReconciliation

ProxySidecarReconciliation component is responsible for keeping the proxy sidecars in the desired state. It restarts Pods that are part of Service Mesh or
that need to be added to the Service Mesh.
The desired state is represented by [Istio CR](../user/03-technical-reference/istio-custom-resource/01-30-istio-custom-resource.md) and [Istio Version](#istio-version).

The following triggers for a restart are be covered by this component:

- Restart Pods with proxy sidecar when CNI config changes.
- Restart Pods with proxy sidecar after an Istio version update.
- Restart Pods with proxy sidecar when proxy resources change.
- Restart Pods if they match [Restart Predicates](#restart-predicates) that the [Istio ResourcesReconciliation component](#istio-resourcesreconciliation) specifies (for example, being up-to-date with EnvoyFilter)

### IngressGatewayReconciler

The IngressGatewayReconciler component is responsible for restarting Istio Ingress Gateway. The component consumes a list of [Restart Predicates](#restart-predicates) that specify when the Ingress Gateway should be restarted.
To understand the reasons for the predicates, refer to the [Architecture Decision Record](https://github.com/kyma-project/istio/issues/278).

## Restart Predicates

The restart predicates are used by the [IngressGatewayReconciler](#ingressgatewayreconciler) and [ProxySidecarReconciliation](#proxysidecarreconciliation) components.
A predicate specifies when the aforementioned components should be restarted. Depending on the implemented interfaces, a predicate can either trigger a restart of Ingress Gateways, Proxy Sidecars or Ingress Gateways and Proxy Sidecars.

For cases where it isn't trivial to check whether the configuration has been applied to the cluster state, a timestamp-based approach is used. For example, the `envoy_filter_allow_partial_referer` resource is annotated with the `istios.operator.kyma-project.io/updatedAt` annotation, which includes the timestamp of its last update.
The predicate initiates a restart of the sidecar and Ingress Gateway if the target had been created prior to this timestamp.
