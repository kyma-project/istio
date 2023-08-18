---
title: Istio sidecars in Kyma and why you want them
---

## Purpose of Istio sidecars

By default, Istio installed as part of the Istio module [is configured](./01-50-default-istio-setup.md) with automatic Istio proxy sidecar injection disabled. This means that none of Pods of your workloads (such as deployments and StatefulSets; except any workloads in the `kyma-system` Namespace) get their own sidecar proxy container running next to your application.

With an Istio sidecar, the resource becomes part of Istio Service Mesh, which brings the following benefits that would be complex to manage otherwise.

## Secure communication

In Kyma's [default Istio configuration](./01-50-default-istio-setup.md), [peer authentication](https://istio.io/latest/docs/concepts/security/#peer-authentication) is set to cluster-wide `STRICT` mode. This ensures that your workload only accepts [mutual TLS traffic](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/) where both, client and server certificates, are validated to have all traffic encrypted. This provides each service with a strong identity, with reliable key and certificate management system.

Another security benefit of having a sidecar proxy is that you can perform [request authentication](https://istio.io/latest/docs/reference/config/security/request_authentication/) for your service. Istio enables request authentication with JSON Web Token (JWT) validation using a custom authentication provider.

## Observability

Furthermore, Istio proxies enhance tracing capabilities by performing global tracing and forwarding the data to a tracing backend using the [OTLP protocol](https://opentelemetry.io/docs/reference/specification/protocol/).

Kiali is another tool that allows you to monitor the service mesh. You can configure Istio to export metrics necessary to support Kiali features that facilitate managing, visualizing, and troubleshooting your service mesh. Follow the [Kiali example](https://github.com/kyma-project/examples/tree/main/kiali) to learn how to deploy Kiali to your Kyma cluster.

By being part of Istio Service Mesh, you can access advanced observability features that would otherwise require complex instrumentation code within your application.

## Traffic management

[Traffic management](https://istio.io/latest/docs/concepts/traffic-management/) is an important feature of service mesh. If you have the sidecar injected into every workload, you can use this traffic management without additional configuration.

With [traffic shifting](https://istio.io/latest/docs/tasks/traffic-management/traffic-shifting/) and [request routing](https://istio.io/latest/docs/tasks/traffic-management/request-routing/), developers can use techniques like canary releases and A/B testing to make their software release process faster and more reliable.

To improve the resiliency of your applications, you can use [mirroring](https://istio.io/latest/docs/tasks/traffic-management/mirroring/) and [fault injection](https://istio.io/latest/docs/tasks/traffic-management/fault-injection/) for testing and audit purposes.

### Resiliency

Application resiliency is an important topic within traffic management. Traditionally, resiliency features like timeouts, retries, and circuit breakers were implemented by application libraries. However, with service mesh, you can delegate such tasks to the mesh, and the same configuration options will work regardless of the programming language of your application. You can read more about it in [Network resilience and testing](https://istio.io/latest/docs/concepts/traffic-management/#network-resilience-and-testing).