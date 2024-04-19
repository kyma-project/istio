# Support for Experimental Configuration Parameters in Istio Custom Resource

## Status
Accepted

## Context
To support features that are not yet production ready, or might not get promoted to a GA feature, we need to introduce a mechanism that would allow users to use these features and provide feedback.

## Decision
Support experimental feature configuration in the Istio custom resource. The features will be exposed under `spec.experimental` path, to ensure that they are separated from GA features.

A sample configuration would look as follows:
```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
spec:
  experimental:
    pilot:
      #[...]
```

The experimental features should be only available to be used with a separately built controller image. Using the experimental features with production image should result in setting the Istio rustom resource to the `Warning` state.

### SAP BTP, Kyma Runtime
In context of SAP BTP, Kyma runtime, experimental features should only be available in the experimental release channel.

## Consequences
We will extend Istio CRD as described in `Decision` section.  A separate image with `{Release-number}-experimental` (for example, `1.0.0-experimental`) will be additionally built apart from the standard, production ready image with `{Release-number}` tag.
The experimental image will be rolled out only in the SAP BTP, Kyma runtime experimental channel. In the fast and regular channels, the production ready image will be rolled out as before.