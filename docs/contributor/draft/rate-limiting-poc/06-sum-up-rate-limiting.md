# Summary of Rate Limiting POC

Istio supports rate limiting by using EnvoyFilter configurations to handle the rate limiting in the Envoy proxies. This can be set to local or global rate limiting. Local rate limiting does not require communication with a rate limit service, and each Envoy instance acts independently. Global rate limiting requires gRPC communication with a rate limit service, for example [envoy ratelimit service](https://github.com/envoyproxy/ratelimit).

Envoy ratelimit service needs to use redis or memcached backend.

# Memcached

Pros:
- Mature project, used in production by many
- Easy to deploy in Kyma cluster
Cons:
- Experimental support in Envoy
- Key size limitation of 250 characters
- Rate limiting based on client's certificate, or very long header values, is not possible

# Redis

Pros:
- Mature and popular
- Supported by Envoy
- Some know how within goat team
Cons:
- Licence issues
- Hyperscaler Redis instance is not accessible from Kyma cluster, or it's very hard to connect to it

# Valkey

Pros:
- Drop-in replacement for Redis
- Fully compatible with Envoys ratelimit service
- Considered stable
- Easy to deploy in Kyma cluster
Cons:
- Not known in the goat team
- Much less popular, and less mature than Redis
