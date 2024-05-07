# Rate Limiting in Istio using Envoy proxies
Istio supports rate limiting by using EnvoyFilter configurations to handle the rate limiting in the Envoy proxies in the Istio Ingress Gateway or
in the service sidecars.

For some scenarios, it might be more efficient to do rate limiting before the traffic reaches the cluster/service mesh. This can be done by using rate limiting capabilities by services offered
by cloud providers like GCP Cloud Armor, AWS WAF, Azure Front Door, etc.

## When should a global or local rate limit be used?
**Global:** If you want to enforce a global access control policy to a particular resource, then global rate limiting should be used. A typical scenario is to set how often a user can access an API according to the user’s service level agreement.

**Local:** A local rate limit is very useful when proxy instances can be horizontally scaled out based on the client load. In this scenario, since each proxy instance get its own rate limit quota, the traffic that the fleet of envoy proxies can handle increase when more proxy instances are spun up.

|                                                 | Global                                                                                                  | Local                                                                                                                                                             |
|-------------------------------------------------|---------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Use Cases                                       | Consistent enforcement, quotas, centralized control                                                     | Resilience, Burst handling                                                                                                                                        |
| Rate Limiting Descriptors                       | Supports Dynamic descriptors, e.g. for header values or Client IPs                                      | Supports only static descriptors                                                                                                                                  |
| Implementation                                  | Centralized Rate Limit Service                                                                          | Each Envoy instance acts independently                                                                                                                            |
| Communication                                   | gRPC communication with Rate limit service adds latency                                                 | No communication                                                                                                                                                  |
| Default rate limiting when no descriptor matches | More complicated, since you need to define a default descriptor, e.g. Client IP, to apply rate limiting | Every rate limiting configuration needs a default token. This can also be a drawback, e.g. if this default is well configured the service would restrict traffic. |

## Common rate limit use cases

- Rate limit for a client
  - based on IP address
  - based on a custom header
  - based on client certificate
- Rate limit for a group of clients
  - based on a custom header
  - based on client certificate
- Rate limit based on a usage tier/plan
  - based on a custom header
- Rate limit to avoid API abuse and brute-force attacks
  - based on a custom header
  - based on IP address
  - based on client certificate
  - based on path
- Rate limit to protect backend services against overload
  - based on a custom header
  - based on IP address
  - based on client certificate
  - based on path

## Rate Limit Enforcement
When introducing rate limiting it's useful to allow a mode where rate limiting behaviour is reflected correctly in the metrics and the response headers, but it's not enforced.
This way you can fine-tune your rate limiting configuration without restricting the clients.

The enablement of this depends on if you are using a local or global rate limiting.

## Rate Limit Metrics
To monitor the rate limiting rules and customise them in a meaningful way, users can use the rate limiting metrics provided by the Envoy instances. When introducing 
rate limiting, we need to provide documentation that enables users to understand and utilise these metrics.

## Rate Limit Response Headers
It's possible to define response headers in the rate limit filter. This can be done using the configuration `response_headers_to_add` for custom headers or use `enable_x_ratelimit_headers` to add headers following the [rate limiting headers proposed in this RFC-draft](https://datatracker.ietf.org/doc/id/draft-polli-ratelimit-headers-03.html).
Additionally, it's also possible to control the retry behaviour for rate limited requests by using the `disable_x_envoy_ratelimited_header` configuration in the rate limit filter.

More information can be found in the [Global Rate Limit Filter](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/ratelimit/v3/rate_limit.proto#rate-limit-proto) and [Local Rate Limit Filter](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/local_ratelimit/v3/local_rate_limit.proto#local-rate-limit-proto) documentation.

## Is there anything to consider when using rate limiting in combination with External Authorizer/Authorization Policies?
In Envoy the rate limit filters are applied before RBAC filters. That means if a rate limiting is happened, the request is not forwarded to the RBAC filters and therefore external authorizer services should not be called.

## Namespace of EnvoyFilter
Depending on if the rate limiting is applied to the ingress gateway or to the service sidecars, the EnvoyFilter should be applied to the respective namespace.
For the service sidecars the EnvoyFilter must be applied to the namespace of the service. For the ingress gateway the EnvoyFilter must be applied to `istio-system`.

## Performance Testing
Executed on Gardener GCP cluster with 3 nodes of n2-standard-4.
For global rate limiting Envoy's reference Rate Limitings Service with a replica count of 3 was used. It's not clear if this is supported by the rate limiting service and if the rate limiting data in Redis is updated correctly, 
but since we have the rate limiting configured in a way that it should never occur, it should not be a problem.
 
From the performance tests, it can be seen that the rate limiting has a significant impact on the request duration. 
While local rate limiting is performing better than global rate limiting, it's still a significant impact on the request duration.
It's interesting that the worst performance is when global rate limiting is applied to the service sidecar. The reason for this might be, that we only have one httpbin instance as load testing service and therefore only one Envoy proxy that
is invoking the rate limiting service. This might be a bottleneck in this scenario.

- No rate limiting
  - [Grafana](https://snapshots.raintank.io/dashboard/snapshot/WqB8kbgM2OylHOFD9xYty6Vodb6hwopJ?orgId=0)
  - [K6 no sidecar summary](./perf_tests/no-rate-limit/summary-no-sidecar.html)
  - [K6 sidecar summary](./perf_tests/no-rate-limit/summary-sidecar.html)
- Local rate limiting on service sidecar
  - [Grafana](https://snapshots.raintank.io/dashboard/snapshot/w2sc5TZWym9OaKBQml991CCTZx2T7cJa?orgId=0)
  - [K6 no sidecar summary](perf_tests/local/rate-limit-on-sidecar/summary-no-sidecar.html)
  - [K6 sidecar summary](perf_tests/local/rate-limit-on-sidecar/summary-sidecar.html)
- Local rate limiting on Ingress Gateway
  - [Grafana](https://snapshots.raintank.io/dashboard/snapshot/CL82OVZPfCFMgCuAvJ5hL8dR4o9xeRFo?orgId=0)
  - [K6 no sidecar summary](perf_tests/local/rate-limit-on-gateway/summary-no-sidecar.html)
  - [K6 sidecar summary](perf_tests/local/rate-limit-on-gateway/summary-sidecar.html)
- Global rate limiting on service sidecar
  - [Grafana](https://snapshots.raintank.io/dashboard/snapshot/q66lxmc9JGvbgoGHOaIzkIdNMAsf7vpP?orgId=0)
  - [K6 no sidecar summary](perf_tests/global/rate-limit-on-sidecar/summary-no-sidecar.html)
  - [K6 sidecar summary](perf_tests/global/rate-limit-on-sidecar/summary-sidecar.html)
- Global rate limiting on Ingress Gateway
  - [Grafana](https://snapshots.raintank.io/dashboard/snapshot/9P53uTH2Ss1yCdtAgtxcPUCXgsf8ADjC?orgId=0)
  - [K6 no sidecar summary](perf_tests/global/rate-limit-on-gateway/summary-no-sidecar.html)
  - [K6 sidecar summary](perf_tests/global/rate-limit-on-gateway/summary-sidecar.html)
- Global rate limiting on Ingress Gateway and local rate limiting on sidecar
  - [Grafana](https://snapshots.raintank.io/dashboard/snapshot/vT3OCvUwzvdKu0iEE3rLEHOWrtaqQiv9?orgId=0)
  - [K6 no sidecar summary](perf_tests/global-local/summary-no-sidecar.html)
  - [K6 sidecar summary](perf_tests/global-local/summary-sidecar.html)

