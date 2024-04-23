# Istio Operator Design

## Status

Accepted

## Context

Since we will be implementing a new Kyma Istio operator to support the new modularised architecture of Kyma, we need to create design documentation to have a better idea of how to implement this new operator.
The technical design is part of the operator documentation and can be found [here](../04-10-technical-design.md).
The functionalities that need to be supported by the new controller can be found in the issue [Technical Design for the Kyma Istio Component](https://github.com/kyma-project/istio/issues/32).

## Decision

Since the most important decisions are also included in the technical design documentation, we will only provide a brief summary in this ADR.

### The Operator Will Use Only One Controller and One Custom Resource
We want to start with a simple implementation and only add more controllers if it is really necessary due to the performance or the reconciliation loop cycle.

### Istio Version
The version of Istio is coupled to the version of the operator. This means that rolling out a new version of Istio requires deploying a new version of the operator.
We chose this way instead of managing the version in the custom resource because we want to have full control over the version and not make the user responsible for managing the versions so that we can also ensure 
that safety-critical versions are rolled out as quickly as possible.

### Istio Installation
The `Install` function of the [Istio Go module] (https://github.com/istio/istio) is used to install Istio as suggested in this [PR](https://github.tools.sap/xf-goat/kyma-istio-operator).  
Although the `Install` function blocks, it is the supported way to install Istio, as it is also used by `istioctl`.  
We also evaluated using the [IstioOperator](https://istio.io/latest/docs/setup/install/operator/) for installation, but we did not think it was a good option for the following reasons:
- Using the operator is discouraged by Istio
- The operator adds a deployment to the cluster that consumes resources, needs to be monitored and can cause security problems.
- The reconciliation operator is also implemented in a blocking way, as it uses the same `HelmReconciler` as the `Install` function.

### Existing Kyma Resources
The `istio-resources` and `certificate` diagrams are split by moving some of them into the new operator, moving some into the `api-gateway operator` and removing some.
We will also remove all dependencies of `global.domainName` from the new operator, so it is easier for us to roll out the first version of it.

### Reconciliation Interval
The synchronisation should be time-based every 5 minutes. To support this, we can use `SyncPeriod` to configure the interval as long as we have only one controller. If we need more control, we should use `RequeueAfter`.
We decided to use 5 minutes to observe how the controller behaves and adjust this value if necessary.

### Controller Components
The controller that performs the reconciliation contains several self-contained components that perform the reconciliation of different resources, e.g. Istio installation, proxy sidecars, Istio ingress gateway.
The components must be self-contained, which means that they can perform the reconciliation independently without relying on the output of other reconciliation components. 
In this way, we can move the independent reconciliation components to a new controller if necessary.

## Consequences
The design is intentionally kept simple to make it easier to follow and to adapt to future requirements.  
It might be necessary to split the controller into several controllers as soon as we have the first insights about the performance or want to perform parts of the reconciliation at different intervals.