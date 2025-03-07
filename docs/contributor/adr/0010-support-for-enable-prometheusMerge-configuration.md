# Support for the Enable PrometheusMerge Configuration

## Status
Approved

## Context
There is a need to support the configuration of enabling `prometheusMerge` to be available in the Istio CR. [Telemetry issue 1468](https://github.com/kyma-project/telemetry-manager/issues/1468) explains the background, and [Telemetry ADR 015](https://github.com/kyma-project/telemetry-manager/blob/main/docs/contributor/arch/015-impact-of-istio-prometheus-merge-on-metric-pipelines.md) explains the impact of enabling this feature.

## Considerations
Based on the [original ADR](https://github.com/kyma-project/telemetry-manager/blob/main/docs/contributor/arch/015-impact-of-istio-prometheus-merge-on-metric-pipelines.md), the original plan was to enable the `prometheusMerge` feature by default. But this has caused problems for users deploying Prometheus with a scrape loop sidecar. For details, see [PR 1184](https://github.com/kyma-project/istio/pull/1184).


## Decision
Enabling the `prometheusMerge` feature must be a configurable option for the users. By default, it will be set to `false`, and users must actively set it to `true` and understand the effects of the feature.

To support this configuration, the following API is proposed in the Istio CR under the `config` option. `prometheusMerge` will have the default value of `false`.
```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
    name: default
    namespace: kyma-system
spec:
 config:
   telemetry:
     metrics:
       prometheusMerge: false
```

The `telemetry` and `metrics` fields are introduced here to account for future plans to introduce more features from the Istio Telemetry API into the CR.

User workloads must be restarted whenever `prometheusMerge` changes, because the Prometheus metrics annotations must be updated with this configuration change.

## Consequences
Istio CustomResourceDefinition will be extended with an additional configuration field of `telemetry.metrics.prometheusMerge`, which supports configuration of the `prometheusMerge` setting in Istio Mesh Config. The field will be an optional configuration with the default value of `false`.

The `telemetry` and `metrics` field will show up in the CR when retrieved from the API server even if not explicitly defined in the manifest.

### Sample Configuration

- The following example uses the default value of `false` for the `prometheusMerge` option:
  ```
  apiVersion: istio.operator.kyma-project.io/v1alpha2
  kind: Istio
  metadata:
    name: default
  
  ```

- The following example sets the value of `prometheusMerge` to `true`:
  ```
  apiVersion: istio.operator.kyma-project.io/v1alpha2
  kind: Istio
  metadata:
    name: default
  spec:
    config:
      telemetry:
        metrics:
          prometheusMerge: true
  ```

Restarts to user workloads happen when the `prometheusMerge` field in the `lastAppliedConfiguration` of Kyma Istio CR module differs from the current `prometheusMerge` in the CR. Additionally, we check and only restart Pods that have incorrect annotations.

If `prometheusMerge` is set to `true`, user workloads are restarted only when they are missing the `prometheus.io/path:/stats/prometheus` and `prometheus.io/port: 15020` annotations. The port value may change depending on the default [statusPort](https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/#ProxyConfig-status_port) configuration.

If `prometheusMerge` is set to `false`, user workloads are restarted only when they have the `prometheus.io/path: /stats/prometheus` and `prometheus.io/port: 15020` annotations.
