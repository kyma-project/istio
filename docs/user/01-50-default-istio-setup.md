---
title: Default Istio setup
---

This document provides an overview of the default setup for Istio. The Istio module uses the [Istio library](https://github.com/istio/istio/tree/master/operator) for installing Istio. The Istio module includes components like Istiod (Pilot) and Ingress Gateway, which are enabled by default.


## Istio module-specific configuration

These configuration changes are applied to customize Istio:

- Both [Istio control plane and data plane](https://istio.io/latest/docs/ops/deployment/architecture/) use distroless images. To learn more, read about [Harden Docker Container Images](https://istio.io/latest/docs/ops/configuration/security/harden-docker-images/).
- Automatic sidecar injection is disabled by default. See how to [enable sidecar proxy injection](./01-60-enable-sidecar-injection.md).
- Resource requests and limits for Istio sidecars are modified to best suit the needs of the evaluation and production profiles.
- [Mutual TLS (mTLS)](https://istio.io/docs/concepts/security/#mutual-tls-authentication) is enabled cluster-wide in the `STRICT` mode.
- Ingress Gateway is expanded to handle ports `80`, `443`, and `31400` for local Kyma deployments.
- The use of HTTP 1.0 is enabled in the outbound HTTP listeners by the `PILOT_HTTP10` flag set in the Istiod component environment variables.
- The configuration file of IstioOperator is modified. [Change Kyma settings](https://kyma-project.io/docs/kyma/latest/04-operation-guides/operations/03-change-kyma-config-values/) to customize the configuration. 