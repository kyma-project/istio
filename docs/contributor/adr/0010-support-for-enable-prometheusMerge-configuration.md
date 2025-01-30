# Support for Enable PrometheusMerge Configuration

## Status
Approved

## Context
There is a need to support the configuration of enabling `prometheusMerge` to be available in the Istio CR. The background behind this need is explained in [this issue](https://github.com/kyma-project/telemetry-manager/issues/1468) and the impact of this feature being enabled was discussed in an [ADR](https://github.com/kyma-project/telemetry-manager/blob/main/docs/contributor/arch/015-impact-of-istio-prometheus-merge-on-metric-pipelines.md) created in the Telemetry Manager repository.

## Considerations
Based on the [original ADR](https://github.com/kyma-project/telemetry-manager/blob/main/docs/contributor/arch/015-impact-of-istio-prometheus-merge-on-metric-pipelines.md), the plan was to originally enable the `prometheusMerge` feature by default. But this has caused problems for users deploying prometheus with a scrape loop sidecar. More details can be found in [this PR](https://github.com/kyma-project/istio/pull/1184).


## Decision
The enablement of the `prometheusMerge` feature needs to be a configurable option for the users. By default it will be set to `false`, but users will then need to actively set it to `true` and understand the effects of the feature.

To allow for this configuration, the following API is proposed in the CR under the config option, `prometheusMerge` will have the default value of `false`.
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

The `telemetry` and `metrics` fields are introduced here to account for future plans to introduce more feature from the Istio Telemetry API into the CR.

## Consequences
Istio CustomResourceDefinition will be extended with an additional configuration field of `telemetry.metrics.prometheusMerge` that will allow for configuration of the `prometheusMerge` setting in Istio Mesh Config. The field will be an optional configuration with the default value of `false`.

The `telemetry` and `metrics` field will show up in the CR when retrieved from the API server even if not explicitly defined in the manifest.

### Sample configuration

1.
```
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default

```
This will use the default value of `false` for the `prometheusMerge` option.

2.
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
This will set the value of `prometheusMerge` to `true`.


