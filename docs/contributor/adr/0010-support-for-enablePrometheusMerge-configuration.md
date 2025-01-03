# Support for Enable Prometheus Merge Configuration

## Status
Proposed

## Context
There is a need to support the configuration of enablePrometheusMerge to be available in the Istio CR. The background behind this need is explained in [this issue](https://github.com/kyma-project/telemetry-manager/issues/1468) and the impact of this feature being enabled was discussed in an [ADR](https://github.com/kyma-project/telemetry-manager/blob/main/docs/contributor/arch/015-impact-of-istio-prometheus-merge-on-metric-pipelines.md) created in the Telemetry Manager repository.

## Considerations
Based on the [original ADR](https://github.com/kyma-project/telemetry-manager/blob/main/docs/contributor/arch/015-impact-of-istio-prometheus-merge-on-metric-pipelines.md), the plan was to originally enable the prometheusMerge feature by default. But this has caused problems for users deploying prometheus with a scrape loop sidecar. More details can be found in [this PR](https://github.com/kyma-project/istio/pull/1184).


## Decision
The enablement of the `prometheusMerge` feature needs to be a configurable option for the users. By default it will be set to `false`, but users will then need to actively set it to `true` and understand the effects of the feature.

To allow for this configuration, the `enablePrometheusMerge` option will be available in the CR under the config option, which has a default value of `false`.

## Consequences
Istio CustomResourceDefinition will be extended with an additional configuration field of enablePrometheusMerge that will allow for configuration of the prometheusMerge setting in Istio Mesh Config. The field will be an optional configuration with the default value of `false`.

### Sample configuration

1.
```
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default

```
This will use the default value of False for the `enablePrometheusMerge` option.

2.
```
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
spec:
  config:
    enablePrometheusMerge: true
```
This will set the value of `enablePrometheusMerge` to `true`.


