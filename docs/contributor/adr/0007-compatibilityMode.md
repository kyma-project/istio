# Support for **compatibilityMode** Configuration

## Status

Accepted

## Context

With Istio's introduction of a [compatibilityVersion](https://istio.io/latest/news/releases/1.21.x/announcing-1.21/#easing-upgrades-with-compatibility-versions) and environment variables allowing to mitigate breaking changes during Istio upgrade, Istio module team decided to introduce compatibility mode to allow users leverage possibility of remaining compatible with previous Istio versions.

## Considerations

Should `compatibilityVersion` be used, exposed by Istio repository with an additional possibility to add environment variables, or a custom implementation allowing Istio team to control the scope of the compatibility settings, either in the environment variables or other parts of the IstioOperator config?

## Decision

To avoid a situation, in which compatibilityVersion introduces changes that have breaking, or unwanted impact on the Istio configuration, Istio team decide not to use `compatibilityVersion`, and to create custom implementatation.

## Consequences

With every Istio major or minor version bump, Istio module team has to manually take care of a proper content of the compatibility settings.

### Sample configuration:

1.
```yaml
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
```
This will set compatibilityMode to false

2.
```yaml
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
spec:
  compatibilityMode: false
```

3.
```yaml
apiVersion: istio.operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
spec:
  compatibilityMode: true
```
