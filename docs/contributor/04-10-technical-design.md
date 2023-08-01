# Technical Design of Istio Operator

This document provides an explanation of the technical design of Istio Operator, including descriptions of each of its components.

## Istio Operator design

![Istio Operator Design](/docs/assets/istio-operator-overview.svg)

Istio Operator contains a single controller composed of several self-contained components, which execute reconciliation logic. To manage Istio, Istio Controller uses [Istio custom resource (CR)](/docs/contributor/04-20-xff-proposal.md), which must be present in the `kyma-system` Namespace.

## Components

Each reconciliation component is completely independent and can determine its actions during the reconciliation without being influenced by the reconciliation of other components. It relies solely on the state within a cluster to make its own calculations.
For the sake of simplicity, the initial design of Istio Operator contains only one controller that handles all the Istio reconciliation logic. Because the components are independent, they can be easily moved into newly created controllers if improving the performance of `IstioController` becomes necessary.

![Controller Component Diagram](/docs/assets/controller-component-diagram.svg)

### IstioController

IstioController is responsible for controlling the entire Istio reconciliation process by triggering the reconciliation components and providing them with the desired state. IstioController is bound to [Istio CR](https://github.com/kyma-project/istio/blob/main/docs/xff-proposal.md). 

### IstioInstallationReconciliation

IstioInstallationReconciliation determines if it is necessary to install, upgrade, or uninstall Istio in a cluster. The component also creates the [IstioOperator CR](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/),
which is used to apply changes to the Istio installation. The applied IstioOperator CR is created by merging `Istio CR` with IstioOperator containing Kyma default values.

The component also detects changes in the `numTrustedProxies` configuration and restarts the Istio Ingress Gateway accordingly. Whenever a change in the `numTrustedProxies` configuration is detected, the Pods in the `istio-system/istio-ingressgateway` deployment are restarted. To determine the current state of the applied Istio configuration, the component reads the `operator.kyma-project.io/lastAppliedConfiguration` annotation from the Istio CR. This annotation is updated after a successful `IstioInstallationReconciliation` run.

To reset a deployment, apply the annotation `istio-operator.kyma-project.io/restartedAt` with a current timestamp in **spec.template.annotations** of the `istio-system/istio-ingressgateway` deployment.

#### IstioClient

IstioClient encapsulates a specific version of the [Istio Go module](https://github.com/istio/istio) and is used to implement calls to Istio API.
Additionally, this component also contains the necessary logic to perform actions such as installing, upgrading, and uninstalling Istio, as described in [Managing Istio with Istio Operator](#installation-of-istio)).

### ProxySidecarReconciliation

ProxySidecarReconcilation component is responsible for keeping the proxy sidecars in the desired state. It restarts Pods that are part of Service Mesh or
that need to be added to Service Mesh.
The desired state is represented by [Istio CR](https://github.com/kyma-project/istio/blob/main/docs/xff-proposal.md) and Istio Version coupled to the Operator.
This component carries a high risk that its execution in this controller takes too long. We need to check its performance during the implementation
and assess whether its logic needs to be executed separately.

For now, the following scenarios must be covered by this component:

- Restart Pods with proxy sidecar when CNI config changes.
- Restart Pods with proxy sidecar after an Istio version update.
- Restart Pods with proxy sidecar when proxy resources change.
- Restart Pods if they match predicates that the IstioResourcesReconciliation component specifies (for example, being up-to-date with EnvoyFilter)

### IstioResoursesReconciliation

IstioResourcesReconciliation is a component responsible for applying resources dependent on Istio, such as VirtualService or EnvoyFilter, and ensuring that the state of the Istio service mesh is configured correctly based on those resources. To maintain the correct state, the IstioResourcesReconciliation component provides predicates on a per-resource basis, which are consumed by the IstioIngressGatewayReconciliation and ProxySidecarReconciliation components. The predicates specify when the aforementioned components should be restarted. 
For cases where it isn't trivial to check whether the configuration has been applied to the cluster state, a timestamp-based approach is used. For example, the envoy_filter_allow_partial_referer resource is annotated with the istios.operator.kyma-project.io/updatedAt annotation, which includes the timestamp of its last update. The predicate initiates a restart of the sidecar and Ingress Gateway if the target had been created prior to this timestamp.

### IstioIngressGatewayReconciliation

The IstioIngressGatewayReconciliation component is responsible for bringing Istio Ingress Gateway to the desired state.