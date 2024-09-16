# Istio Sidecar Proxies

A service mesh is an infrastructure layer that handles service-to-service communication, proxying, service discovery, traceability, and security, independently of the code of the services. To deliver this functionality, the Istio module uses the [Istio service mesh](https://istio.io/docs/concepts/what-is-istio/) that is customized for the specific needs of an implementation.

The main principle of the Istio service mesh is to inject Pods of every service with Istio sidecar proxy, which is an extended version of the Envoy proxy. Envoy intercepts the communication between the services and regulates it by applying and enforcing the rules you create.

## Purpose and Benefits of Istio Sidecars

By default, Istio installed as part of the Istio module is configured with automatic Istio sidecar proxy injection disabled. This means that none of your workloads' Pods, except those in the `kyma-system` namespace, get their own sidecar proxy container running next to the application. When sidecar injection is disabled for a service or for a namespace, you must traffic configuration in services or at a namespace level by creating [DestinationRule](https://istio.io/docs/reference/config/networking/destination-rule/) and [PeerAuthentication](https://istio.io/docs/tasks/security/authentication/authn-policy/) resources.

With an Istio sidecar proxy injected, a resource becomes part of the Istio service mesh, which brings the following benefits that would be complex to manage otherwise.

### Secure Communication
<!-- markdown-link-check-disable-next-line -->
The Istio module sets [peer authentication](https://istio.io/latest/docs/concepts/security/#peer-authentication) to cluster-wide `STRICT` mode. This ensures that your workload only accepts [mutual TLS traffic](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/) where both client and server certificates are validated to ensure that all traffic is encrypted. This provides each service with a strong identity and a reliable system for managing keys and certificates.

Another security benefit of having a sidecar proxy is that you can perform [request authentication](https://istio.io/latest/docs/reference/config/security/request_authentication/) for your service. Istio enables request authentication with JSON Web Token (JWT) validation using a custom authentication provider.

### Observability

Istio proxies enhance tracing capabilities by performing global tracing and forwarding the data to a tracing backend using the [OTLP protocol](https://opentelemetry.io/docs/reference/specification/protocol/). By being part of the Istio service mesh, you can access advanced observability features that would otherwise require complex instrumentation code within your application.

### Traffic Management

If you have the sidecar injected into every workload, you can use Istio’s traffic routing rules without additional configuration. See [Traffic management](https://istio.io/latest/docs/concepts/traffic-management/).

[Traffic shifting](https://istio.io/latest/docs/tasks/traffic-management/traffic-shifting/) and [request routing](https://istio.io/latest/docs/tasks/traffic-management/request-routing/) allows you to use techniques like canary releases and A/B testing to make your software release process faster and more reliable. To improve the resiliency of your applications, you can use [mirroring](https://istio.io/latest/docs/tasks/traffic-management/mirroring/) and [fault injection](https://istio.io/latest/docs/tasks/traffic-management/fault-injection/) for testing and audit purposes.

### Resiliency

Application resiliency is an important topic within traffic management. Traditionally, resiliency features like timeouts, retries, and circuit breakers were implemented by application libraries. However, with service mesh, you can delegate such tasks to the mesh, and the same configuration options will work regardless of the programming language of your application. See [Network Resilience and Testing](https://istio.io/latest/docs/concepts/traffic-management/#network-resilience-and-testing).

## Restart of Workloads with Enabled Istio Sidecar Injection

When the Istio version is updated or the configuration of the proxies is changed, the Pods that have Istio injection enabled are automatically restarted. This is possible for all resources that allow for a rolling restart. If Istio is uninstalled, the workloads are restarted again to remove the sidecars.
However, if a resource is a Job, a ReplicaSet that is not managed by any Deployment, or a Pod that is not managed by any other resource, the restart cannot be performed automatically. In such cases, a warning is logged, and you must manually restart the resources.
The Istio module does not restart an Istio sidecar proxy, if it has a custom image set. See [Resource Annotations](https://istio.io/latest/docs/reference/config/annotations/#SidecarProxyImage).