# Summary of Rate Limiting POC

Istio supports rate limiting by using EnvoyFilter configurations to handle the rate limiting in the Envoy proxies. This can be set to local or global rate limiting. Local rate limiting does not require communication with a rate limit service, and each Envoy instance acts independently. Global rate limiting requires gRPC communication with a rate limit service, for example [envoy ratelimit service](https://github.com/envoyproxy/ratelimit).

Envoy ratelimit service needs to use redis or memcached backend.

# Memcached

Memcached support is experimental, usage in production may not be recommended. It also has a key size limitation of 250 characters, therefore rate limiting based on client's certificate, or very long header values, is not possible. It cannot be considered as a viable option.

# Redis

Redis is considered stable, and is the recommended backend for rate limiting. Due to Redis licence we cannot deploy it in kyma cluster, and for now it's not possible (or very hard) to connect to BTP hyperscaler Redis instance from the Kyma cluster. We need to discuss the possibility of having redis managed by btp-operator.

# Valkey

Valkey is a drop-in replacement for Redis, fully compatible with Envoys ratelimit service, considered stable, it does not have the key size limitation. It is also possible to deploy it in the Kyma cluster. Currently, it might be the only working option for rate limiting in Kyma.
