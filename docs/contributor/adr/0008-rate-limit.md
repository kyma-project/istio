# Rate Limit

## Status

Proposed

## Context

It has been decided to introduce rate limit functionality into Kyma to allow users to set rate limit functionality on the service mesh layer, therefore allowing to consume intended service mesh functionality which is abstracting away networking concerns outside applications inside the mesh.
Since Istio is an underlying service mesh responsible for the workload networking across Kyma clusters, it is Istio that allows to configure such a functionality.

In Envoy, there is support for local and global rate limiting. The major difference between local and global rate limiting is that local rate limiting is applied directly at the sidecar proxy level, controlling the rate of requests on a per-instance basis. 
For consistent rate limiting across your service mesh, the global rate limiting can be utilised.  
The global rate limiting requires a rate limit service that adheres to the [gRPC RateLimit v3 protocol](https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/ratelimit/v3/rls.proto). This centralised approach enforces shared stateful rate limits, applicable to all instances and services within the mesh.
Currently, there are limitations that prevent using Redis on managed Kyma clusters. Because global rate limiting relies on Redis for persistence, the global rate limiting is not in scope of this ADR.

## Decision

New CRD will be introduced in the Istio module to allow rate limiting configuration.
In the future, global rate limit might be introduced. The decision has not been made yet, so it has been decided to name the new CRD `RateLimit` not to indicate that global rate limit will be provided. By naming the current CRD, for example, `LocalRateLimit,` logical would be to expect also `GlobalRateLimit` to exist at some point.  
Currently, the rate limit functionality can only restrict the number of requests without any other criteria like header presence or header value. 
The CRD is structured in a way that allows further extensions, such as configuring different rate limit criteria or introducing global rate limiting.

### Rate Limit Metrics
The [Envoy LocalRateLimit filter](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/local_ratelimit/v3/local_rate_limit.proto#local-rate-limit-proto) requires the field **stat_prefix** to be set. This field is used to generate the metrics for the rate limit.  
The decision is not to expose the **stat_prefix** field to the user. Additionally, the field will be set to the value `rate_limit` for now because we have very limited configuration options in the Rate Limit CR.

The metrics are generated in the following format:
```
rate_limit.http_local_rate_limit.enabled: 6 
rate_limit.http_local_rate_limit.enforced: 0 
rate_limit.http_local_rate_limit.ok: 4 
rate_limit.http_local_rate_limit.rate_limited: 2 
```

If we introduce more configuration options in the RateLimit CR in the future, we might consider exposing the **stat_prefix** in RateLimit CR or automatically derive additional **stat_prefix** values from the configuration, for example, `rate_limit.by_header`.


### Enforcing Rate Limit
The [Envoy LocalRateLimit filter](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/local_ratelimit/v3/local_rate_limit.proto#local-rate-limit-proto) allows to configure the enforcement for 0 - 100% percentage of the traffic by using the field **filter_enforced**.   
It is considered a good use case to allow to create a RateLimit CR without enforcing the rate limit. By following this approach, it is possible to check the rate limit metrics to understand the impact on real traffic before blocking it.  
The decision is to hide the complexity of the Envoy enforcement configuration behind the required boolean field **spec.enforce**, so the user must decide whether to enforce the rate limit.

### Scope of the RateLimit CR

When there's a requirement to enforce rate limiting across all namespaces, the RateLimit CR should be deployed within the `istio-system` namespace. This approach aligns with the strategy used for EnvoyFilters when applying configurations globally. Consequently, the associated EnvoyFilter must also be located in the `istio-system` namespace. In this context, the **spec.workloadSelector** field should be omitted from the RateLimit CR.  
For imposing rate limiting on every workload or Istio Ingress Gateway within a specific namespace, the RateLimit CR needs to be applied to the targeted namespace itself. Under these circumstances, the **spec.workloadSelector** field should also be excluded.
Depending on the configuration of **spec.context** the rate limit will be applied to the workload sidecars or the Istio Ingress Gateway.

### Rate Limit HTTP Headers

While enabling rate limit feature on the HTTP layer, there is a possibility to give insights to the client about the current rate limiting state e.g. how many requests are left to be done before being rate limited, or how much time is left until rate limit tokens are refilled. 
The decision is that the user can enable rate limit headers in the RateLimit CR by setting the boolean field **spec.enableHeaders**. Following security good practices, these headers are disabled by default to limit internal information exposure.

### RateLimit CR Spec

| field                      | type                | description                                                                                                                                                                                                                                               | required |
|----------------------------|---------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|
| context                    | string              | Allows to set the context for the underlying EnvoyFilter. The value can be `gateway`, which means the rate limit is applied to the Istio Ingress Gateways, or `sidecar`, which means it is applied to sidecar proxies of the workloads.                            | yes      |
| enableHeaders              | boolean             | Enables **x-rate-limit** response headers. The default value is `false`.                                                                                                                                                                                         | yes      |
| enforce                    | boolean             | Allows to choose if the rate limit should be enforced.                                                                                                                                                                                                    | yes      | 
| workloadSelector           | object              | If omitted, the rate limit is applied to all workload instances in the same namespace as the RateLimit CR. If the RateLimit CR is present in the `istio-system` namespace, it is applied to all workloads in all namespaces.                       | no       |
| workloadSelector.labels    | map<string, string> | One or more labels that indicate a specific set of Pods or VMs on which the configuration should be applied. The scope of label search is restricted to the namespace in which the resource is present.                                    | no       |
| local                      | object              | Allows to describe local rate limit properties.                                                                                                                                                                                                            | yes      |
| local.bucket               | object              | The token bucket configuration to use for rate limiting requests. Each request consumes a single token. If the token is available, the request will be allowed. If no tokens are available, the request will be rejected with status code `429`.            | yes      |
| local.bucket.maxTokens     | int                 | The maximum tokens that the bucket can hold. This is also the number of tokens that the bucket initially contains.                                                                                                                                        | yes      |
| local.bucket.tokensPerFill | int                 | The number of tokens added to the bucket during each fill interval.                                                                                                                                                                                       | yes      |
| local.bucket.fillInterval  | duration            | The fill interval that tokens are added to the bucket. During each fill interval, `tokensPerFill` are added to the bucket. The bucket will never contain more than `maxTokens` tokens. The `fillInterval` must be greater than or equal to 50ms to avoid excessive refills. | yes      |

## Usage Example

```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: RateLimit
metadata:
  name: httpbin-local-rate-limit
  namespace: default
spec:
  context: "sidecar" # should be on this level to not mix sidecar level rate limit with gateway level rate limit. In such case, 2 separate resources should be created.
  enableHeaders: true
  workloadSelector:
    labels:
      app: httpbin
  local:
    bucket:
      maxTokens: 1000
      tokensPerFill: 1000
      fillInterval: 1s
```

The following diagram illustrates Istio Controller technical design extended with the RateLimit controller loop.

![Istio Controller overview with RateLimit](../../assets/istio-operator-overview-ratelimit.svg)


## Consequences

New controller for the new RateLimit CRD needs to be implemented as a part of the Istio module.
The new RateLimit CR can be used to set rate limits in the cluster's service mesh without having to worry about possible changes in the Istio EnvoyFilter resources.
Nevertheless, EnvoyFilter is complex and since its API is not stable yet, the current RateLimit CR will be provided in `v1alpha1` version and might be changed in the future Istio releases.
The RateLimit CRD should be included in the blocking deletion strategy for Istio CR since Istio should not be uninstalled while RateLimit CRs exist in the cluster.
Requests beyond the allowed rate limit threshold will get the HTTP `429` response.
The new functionality requires integration tests to be implemented.
