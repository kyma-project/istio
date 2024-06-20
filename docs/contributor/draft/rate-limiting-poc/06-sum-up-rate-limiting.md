# Summary of Rate Limiting POC

Istio supports rate limiting by using EnvoyFilter configurations to handle the rate limiting in the Envoy proxies. This can be set to local or global rate limiting. Local rate limiting does not require communication with a rate limit service, and each Envoy instance acts independently. Global rate limiting requires gRPC communication with a rate limit service. Envoy provides a [reference implementation](https://github.com/envoyproxy/ratelimit) written in Go. The Envoy rate limit service reference implementation needs to use Redis or Memcached backend.

Additional information about Envoy rate limit service usage:
- Production use at Lyft [for over 2 years](https://github.com/envoyproxy/ratelimit?tab=readme-ov-file#api-deprecation-history)
- Gloo Edge by solo.io [also uses the reference implementation for rate limiting](https://docs.solo.io/gloo-edge/latest/guides/security/rate_limiting/)
- Tinder tried a few other solutions [before switching to Envoy rate limit service](https://www.youtube.com/watch?v=2EKU8zCQAow)

The rate limit service had last release over four years ago, but it is still maintained and built on the main branch.

All the backends listed here are easy to deploy using Helm charts. 

# Memcached

Pros:
- Mature project, used in production by many
- Fully open source, no licensing issues
Cons:
- Only experimental support in [Envoy rate limit service implementation](https://github.com/envoyproxy/ratelimit?tab=readme-ov-file#memcache), no further information available
- Descriptor key size limitation of 250 characters
- Rate limiting based on the client's certificate or very long header values is not possible

# Redis
Redis is offered in BTP as a service, making it best option for our needs. Unfortunately it is not accessible from the Kyma cluster.

Pros:
- Mature and popular
- Supported by Envoy rate limit service implementation
- There are people in the Goat team who worked with Redis in the past
Cons:
- License prevents us from offering it as a service in Kyma
- Hyperscaler Redis instance is not accessible from Kyma cluster, it is only available within Cloud Foundry environment
- If we choose the managed Redis offering, it will not be available for the open-source Istio module

# [Valkey](https://github.com/valkey-io/valkey)
Valkey is an open source, in memory datastore released under the BSD-3 Clause License. It is a continuation of the work that was being done on Redis 7.2.4.

Pros:
- Fully compatible with Envoy's rate limit service, according to our usage scenarios
- Stable project [supported by the Linux Foundation](https://www.linuxfoundation.org/press/linux-foundation-launches-open-source-valkey-community)
Cons:
- Not known in the Goat team
- Less popular than Redis
