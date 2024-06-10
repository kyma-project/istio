# Support for **compatibilityMode** Configuration

## Status

Accepted

## Context

With Istio's introduction of a [compatibilityVersion](https://istio.io/latest/news/releases/1.21.x/announcing-1.21/#easing-upgrades-with-compatibility-versions) and the environment variables allowing to mitigate breaking changes during Istio upgrade, Istio module team decided to introduce compatibility mode to allow users leverage possibility of remaining compatible with previous Istio versions.

## Decision

To avoid a situation, in which compatibilityVersion introduces changes that have a breaking, or an unwanted impact on the Istio configuration, Istio team decided to not to use the `compatibilityVersion`, and to create a custom implementatation. The custom implementation in the Istio Module controller applies a subset of the Istio compatibilityVersion flags that are examined and extracted by the Istio team, and that fits with the configuration provided by the Istio Module. This decision was made, because Istio compatibilityVersion 1.21 would introduce configuration that was on purpose already removed from the IstioOperator before, and there is no guarantee that the following compatibityVersion of the Istio in the future, would not include breaking changes for the Istio Module opinionated configuration. Since compatibility support is meant for one minor version back, the compatibility mode from the user perspective has been decided to be a boolean flag on the IstioCR.

## Consequences

With every Istio major or minor version bump, Istio module team has to manually take care of an examination of the compatibility flags provided by the version of the Istio. Also the Istio Module implementation has to be adjusted depending on the scope of the compatibilityVersion for the given Istio version.

