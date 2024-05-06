# Rate Limiting in Istio using Envoy proxies
Istio supports rate limiting by using EnvoyFilter configurations to handle the rate limiting in the Envoy proxies in the Istio Ingress Gateway or
in the service sidecars.

For some scenarios, it might be more efficient to do rate limiting before the traffic reaches the cluster/service mesh. This can be done by using rate limiting capabilities by services offered
by cloud providers like GCP Cloud Armor, AWS WAF, Azure Front Door, etc.

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

## When should a global or local rate limit be used?
**Global:** If you want to enforce a global access control policy to a particular resource, then global rate limiting should be used. A typical scenario is to set how often a user can access an API according to the user’s service level agreement.

**Local:** A local rate limit is very useful when proxy instances can be horizontally scaled out based on the client load. In this scenario, since each proxy instance get its own rate limit quota, the traffic that the fleet of envoy proxies can handle increase when more proxy instances are spun up.

TLDR comparsion:

|                                                  | Global                                                                                                  | Local                                                                                                                                                             |
|--------------------------------------------------|---------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Use Cases                                        | Consistent enforcement, quotas, centralized control                                                     | Resilience, Burst handling                                                                                                                                        |
| Rate Limiting Descriptors                        | Supports Dynamic descriptors, e.g. for header values or Client IPs                                      | Supports only static descriptors                                                                                                                                  |
| Implementation                                   | Centralized Rate Limit Service                                                                          | Each Envoy instance acts independently                                                                                                                            |
| Communication                                    | gRPC communication with Rate limit service adds latency                                                 | No communication                                                                                                                                                  |
| Default rate limiting when no descriptor matches | More complicated, since you need to define a default descriptor, e.g. Client IP, to apply rate limiting | Every rate limiting configuration needs a default token. This can also be a drawback, e.g. if this default is well configured the service would restrict traffic. |


