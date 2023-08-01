# Ownership of resources in the Kyma repository

In order to transition to a more modularised architecture, the [IstioOperator resource](https://github.com/kyma-project/kyma/tree/main/resources/istio),
the [additional istio-resources](https://github.com/kyma-project/kyma/tree/main/resources/istio-resources), and
the [certificates](https://github.com/kyma-project/kyma/tree/main/resources/certificates) must be moved to the new modules.

## Istio Operator resource

The Istio Operator resource is moved into the new Kyma Istio Operator. It is used to define default values for Istio, which the user can customise by modyfying Istio CR.

## Istio resources

### Istio Grafana dashboards

It still needs to be decided who will have ownership of the dashboards. To make the right choice, such aspects as the change interval or relevance of Istio version updates should be considered.

### Istio ServiceMonitor

Istio ServiceMonitor is planned to be replaced. For more information, see this [PR](https://github.com/kyma-project/kyma/pull/16247).

### istio-healthz Virtual Service

Isito-healthz Virtual Service offers the possibility of monitoring Istio externally by exposing an endpoint. This resource is not part of the Istio module.
Therefore, if a user wants to enable external monitoring, they must configure it separately.

### Global mTLS PeerAuthentication

Global mTLS PeerAuthentication is tightly coupled with the Istio installation. Therefore, Istio Operator is responsible for reconciling this resource.

### Kyma Gateway

Kyma Gateway is moved to the API Gateway, as it is a default gateway we provide, and its responsibilities are more closely connected to API Gateway than to Istio. Since API Gateway is already dependent on Istio, moving it does not create any additional dependency.

## Certificate resources

Certificate resources are moved to API Gateway, as they are tightly coupled with the Kyma Gateway resource.