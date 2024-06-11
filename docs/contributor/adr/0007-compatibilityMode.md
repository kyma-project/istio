# Support for **compatibilityMode** Configuration

## Status

Accepted

## Context

With Istio's introduction of [compatibilityVersion](https://istio.io/latest/news/releases/1.21.x/announcing-1.21/#easing-upgrades-with-compatibility-versions) and the environment variables allowing to mitigate breaking changes during an Istio upgrade, the Istio module team decided to introduce compatibility mode to allow users leverage possibility of remaining compatible with previous Istio versions.

## Decision

To avoid a situation, in which compatibilityVersion introduces changes that have a breaking or an unwanted impact on the Istio configuration, the Istio team decided not to use the **compatibilityVersion** and create a custom implementation instead. The custom implementation in the Istio module's controller applies a subset of the Istio compatibilityVersion flags that are examined and extracted by the Istio team, and that fit with the configuration provided by the Istio module. This decision was made because Istio compatibilityVersion 1.21 would introduce a configuration that had intentionally been removed from Istio Operator before. Additionally, there is no guarantee that future compatibility versions of Istio would not include breaking changes for the Istio module's opinionated configuration. Since compatibility support is only meant for one minor version back, from the use perspective, it can be set as a boolean flag in the Istio CR.

## Consequences

With every major or minor version bump, the Istio module team has to manually examine the compatibility flags provided by the Istio version. Also, the Istio module's implementation has to be adjusted depending on the scope of the compatibilityVersion for the given Istio version.

