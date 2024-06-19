# Summary of Rate Limiting POC

Istio supports rate limiting by using EnvoyFilter configurations to handle the rate limiting in the Envoy proxies. This can be set to local or global rate limiting. Local rate limiting does not require communication with a rate limit service, and each Envoy instance acts independently. Global rate limiting requires gRPC communication with a rate limit service. Envoy provides a [reference implementation](https://github.com/envoyproxy/ratelimit) written in Go.

The Envoy rate limit service reference implementation needs to use Redis or Memcached backend.

# Memcached

Pros:
- Mature project, used in production by many
- Easy to deploy in a Kyma cluster
Cons:
- Only experimental support in Envoy
- Descriptor key size limitation of 250 characters
- Rate limiting based on the client's certificate or very long header values is not possible

# Redis

Pros:
- Mature and popular
- Supported by Envoy
- There are people in the Goat team who worked with Redis in the past
Cons:
- Licence issues
- [Hyperscaler Redis instance is not accessible from Kyma cluster](https://sap-btp.slack.com/archives/C01LGCBS196/p1718107858028479?thread_ts=1718018170.520259&cid=C01LGCBS196)

# Valkey

Pros:
- Drop-in replacement for Redis
- Fully compatible with Envoy's rate limit service
- Considered stable
- Easy to deploy in a Kyma cluster
Cons:
- Not known in the Goat team
- Much less popular and less mature than Redis
