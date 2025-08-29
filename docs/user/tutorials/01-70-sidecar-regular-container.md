# Configure Istio Proxy as a Regular Container

## Context

Istio since version 1.27 changed the default behavior of the Istio proxy injection to use an initContainer instead of a regular container. You can read about benefits of this approach on the official [Istio blog](https://istio.io/latest/blog/2023/native-sidecars/).
However, in some cases, you might want to revert to the previous behavior and have the sidecar as a regular container.

## Solution

To set the Istio proxy to be a regular container, you can use the following [annotation](https://istio.io/latest/docs/reference/config/annotations/#SidecarNativeSidecar) with your Pod:

```yaml
sidecar.istio.io/nativeSidecar: false
```