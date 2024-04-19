# Istio Resources Reconciliation

## Status
Accepted

## Context
We need to support configuration for resources that depend on Istio CRDs, which have been already installed on the cluster. To address this, we create an `IstioResourcesReconciliation` component that will ensure that the state of these resources is up-to-date.

## Decision
Create the `IstioResourcesReconciliation` component. The component will define the resources (in the form of `YAML` documents) and provide predicates on a per-resource basis. These predicated will trigger a restart of the Istio Ingress gateway or/and a restart of the Istio proxy.

## Consequences
The possibility of creating resources that require the presence of CRDs installed by Istio install becomes available in Istio Controller.

For the `envoy_filter_allow_partial_referer` resource, we will use a timestamp-based approach to check if the configuration has been propagated to the service mesh. To achieve this, we will annotate the EnvoyFilter resource with the `istios.operator.kyma-project.io/updatedAt` annotation containing the timestamp of its last update. This approach is necessary because checking proxy configuration is a non-trivial and potentially dangerous procedure, which requires making an HTTP request to every sidecar's `/config_dump:15000` endpoint. This is due to the fact that `istiod` does not provide the actual status for the `bootstrap` envoy configuration.

The timestamp-based approach may also be used for other resources.