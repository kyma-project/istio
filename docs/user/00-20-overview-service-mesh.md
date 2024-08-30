# Istio Service Mesh

## What is the Istio service mesh?

A service mesh is an infrastructure layer that handles service-to-service communication, proxying, service discovery, traceability, and security, independently of the code of the services. To deliver this functionality, the Istio module uses the [Istio](https://istio.io/docs/concepts/what-is-istio/) service mesh that is customized for the specific needs of the implementation.

The main principle of the Istio service mesh is to inject Pods of every service with the Envoy sidecar proxy. Envoy intercepts the communication between the services and regulates it by applying and enforcing the rules you create.

## Purpose and Benefits of Istio Sidecars

By default, Istio installed by Istio Operator is configured with automatic Istio sidecar proxy injection disabled. This means that none of the Pods of your workloads, except any workloads in the `kyma-system` namespace, get their own sidecar proxy container running next to your application. You can manage mTLS traffic in services or at a namespace level by creating [DestinationRule](https://istio.io/docs/reference/config/networking/destination-rule/) and [PeerAuthentication](https://istio.io/docs/tasks/security/authentication/authn-policy/) resources. If you disable sidecar injection for a service or for a namespace, you must manage their traffic configuration by creating appropriate DestinationRule and PeerAuthentication resources.

With an Istio sidecar, the resource becomes part of the Istio service mesh, which brings the following benefits that would be complex to manage otherwise.

### Secure Communication
<!-- markdown-link-check-disable-next-line -->
In Kyma's [default Istio configuration](./00-40-overview-istio-setup.md), [peer authentication](https://istio.io/latest/docs/concepts/security/#peer-authentication) is set to cluster-wide `STRICT` mode. This ensures that your workload only accepts [mutual TLS traffic](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/) where both client and server certificates are validated to ensure that all traffic is encrypted. This provides each service with a strong identity and a reliable system for managing keys and certificates.

Another security benefit of having a sidecar proxy is that you can perform [request authentication](https://istio.io/latest/docs/reference/config/security/request_authentication/) for your service. Istio enables request authentication with JSON Web Token (JWT) validation using a custom authentication provider.

### Observability

Istio proxies enhance tracing capabilities by performing global tracing and forwarding the data to a tracing backend using the [OTLP protocol](https://opentelemetry.io/docs/reference/specification/protocol/). By being part of the Istio service mesh, you can access advanced observability features that would otherwise require complex instrumentation code within your application.

### Traffic Management

[Traffic management](https://istio.io/latest/docs/concepts/traffic-management/) is an important feature of service mesh. If you have the sidecar injected into every workload, you can use Istio’s traffic routing rules without additional configuration.

With [traffic shifting](https://istio.io/latest/docs/tasks/traffic-management/traffic-shifting/) and [request routing](https://istio.io/latest/docs/tasks/traffic-management/request-routing/), developers can use techniques like canary releases and A/B testing to make their software release process faster and more reliable.

To improve the resiliency of your applications, you can use [mirroring](https://istio.io/latest/docs/tasks/traffic-management/mirroring/) and [fault injection](https://istio.io/latest/docs/tasks/traffic-management/fault-injection/) for testing and audit purposes.

### Resiliency

Application resiliency is an important topic within traffic management. Traditionally, resiliency features like timeouts, retries, and circuit breakers were implemented by application libraries. However, with service mesh, you can delegate such tasks to the mesh, and the same configuration options will work regardless of the programming language of your application. See [Network Resilience and Testing](https://istio.io/latest/docs/concepts/traffic-management/#network-resilience-and-testing).

