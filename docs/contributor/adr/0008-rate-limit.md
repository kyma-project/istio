# Rate Limit

## Status

Proposed

## Context

It has been decided to introduce rate limit functionality into Kyma platform to allow users to set rate limit functionality on the service mesh layer, therefore allowing to consume intended service mesh functionality which is abstracting away networking concerns outside applications inside the mesh.
Since Istio is an underlying service mesh responsible for the workloads networking across Kyma clusters, therefore it is the Istio which allows to configure such functionality.
In the Envoy, there is a support for local and global rate limiting. The major difference between local and global rate limit is that local rate limiting is applied directly at the sidecar proxy level, controlling the rate of requests on a per-instance basis. 
For consistent rate limiting across your service mesh, the global rate limiting can be utilised. The global rate limiting requires a rate limit service that adheres to the [gRPC RateLimit v3 protocol](https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/ratelimit/v3/rls.proto). This centralised approach enforces shared stateful rate limits, applicable to all instances and services within the mesh.
Currently, there are limitations that prevent using Redis on managed Kyma clusters. Because global rate limiting relies on Redis for persistence, the global rate limiting is not in scope of this ADR.

## Decision

New CRD will be introduced in the Istio module to allow rate limiting configuration.
In the future, global rate limit might be introduced. The decision has not been made yet, therefore it has been decided to name the new CRD `RateLimit` to not to indicate, that global rate limit will be provided. By naming current CRD for example `LocalRateLimit` logical would be to expect also `GlobalRateLimit` to be existent at some point.
Currently, the possibilities for the rate limit functionality are limited to restrict number of requests, without any other criteria like header presence or header value. 
CRD is structured in a way that allows further extensions, allowing to configure different rate limit criteria, or to introduce global rate limit.

### Rate Limit Metrics
The [Envoy LocalRateLimit filter](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/local_ratelimit/v3/local_rate_limit.proto#local-rate-limit-proto) requires the field `stat_prefix` to be set. This field is used to generate the metrics for the rate limit.  
The decision is to not expose the `stat_prefix` field to the user. Additionally, the field will be set to the value `rate_limit` for now, because we have very limited configuration options in the Rate Limit CR.   

The metrics are generated in the following format:
```
rate_limit.http_local_rate_limit.enabled: 6 
rate_limit.http_local_rate_limit.enforced: 0 
rate_limit.http_local_rate_limit.ok: 4 
rate_limit.http_local_rate_limit.rate_limited: 2 
```

If we introduce more configuration options in RateLimit CR the future, we might consider to expose the `stat_prefix` in RateLimit CR or automatically derive additional `stat_prefix` values from the configuration, e.g. `rate_limit.by_header`.


### Enforcing Rate Limit
The [Envoy LocalRateLimit filter](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/local_ratelimit/v3/local_rate_limit.proto#local-rate-limit-proto) allows to configure the enforcement for 0 - 100% percentage of the traffic by using the field `filter_enforced`.   
It is considered a good use case to allow to create a RateLimit without enforcing the rate limit. By following this approach it is possible to check the rate limit metrics to understand the impact on real traffic, before blocking it.  
The decision is to hide the enforcement configuration behind the required bool field `enforced`, to make it easier for the user to understand the configuration.

## Scope of the rate limiting

In case of need to apply RateLimit CR to all namespaces, it should be placed in the istio-system, mimicking EnvoyFilter strategy for global application. Therefore underlying EnvoyFilter should be placed in the istio-system namespace as well. In such scenario spec.workloadSelector should not be present.
In case of applying RateLimit CR to the whole namespace, it should be placed in the namespace to which it should apply. In such scenario spec.workloadSelector should not be present.

## Rate limit HTTP headers

While enabling rate limit feature on the HTTP layer, there is a possibility to give insights to the client about the current rate limitting state e.g. how many requests are left to be done before being timed out, or how much time is left to reset the rate limit capacity. As a desicion in this scope it is that it will be enabled for the users to voluntarily opt in for such headers, by setting boolean field in the RateLimit CR. It has been decided to turn off by default those headers as a security best practice, since such headers despite being useful, could be used to fine tune DoS type of attacks, making it harder to detect. Also through those headers, insights might be get on the requests scale getting into the system.

## Consequences

New controller for the new rate limit CRD needs to be implemented as a part of the Istio module.
The new rate limit CR can be used by users to set rate limits in the service mesh in their cluster without having to worry about possible changes in the Istio EnvoyFilter resources.
Nevertheless EnvoyFilter is complex and since it's API is not stable yet, current RateLimit CR will be provided as v1alpha1 version, and might be changed in the future Istio releases.
RateLimit CRD should be included into blocking deletion strategy for Istio CR, since Istio should not be uninstalled while RateLimit CRs exist in the cluster.
Requests beyond the allowed rate limit capacity will get HTTP 429 response.
New functionality requires integration tests to be implemented.
