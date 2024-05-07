# Local Rate Limiting

For local rate limiting, you can use a global rate limiting configuration on the http filter level as described in the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter#id1).
Since we need to configure custom descriptors, the [global rate limiting configuration must be disabled](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter#id2) and [Rate Limit Actions](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route_components.proto#config-route-v3-ratelimit-action) and [descriptors](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter#using-rate-limit-descriptors-for-local-rate-limiting) can be configured for 

There is some special handling for descriptors that needs to be considered:
- If there is no matching descriptor entries, the global token bucket on the Rate Limit Filter of the Route is used.
- All the matched local descriptors will be sorted by tokens per second and the tokens are consumed in that order.
- Global tokens are not sorted, so it's suggested that they should be larger than other descriptors.

## Limitations
The Envoy local rate limiting is limited, since it seems that some RateLimitActions are not supported in the same way as for global rate limiting, and it's not possible to enforce rate limiting across multiple instances of the same service.
Known restrictions for RateLimitActions that behaves different for local rate limiting:
- The descriptor handling for [remote_address](https://github.com/envoyproxy/envoy/blob/10a10039b3a82d43ff47c319e0ef4faf229f3327/source/common/router/router_ratelimit.cc#L143-L143) is implemented in Envoy. The issue is that the IP is not extracted from the XFF header when using local rate limiting, and you have to provide a value that is used for matching in the descriptor.
- The `header_request` RateLimitAction needs a descriptor value to match against, which is not needed for global rate limiting.

There is a [GitHub issue](https://github.com/envoyproxy/envoy/issues/19895) for Envoy to support dynamic descriptors for local rate limiting.

## Rate Limit Enforcement
The [LocalRateLimit filter configuration](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/local_ratelimit/v3/local_rate_limit.proto#local-rate-limit-proto) `filter_enforced` can be used to configure the enforcement of the rate limiting.

## Example scenarios

### Setup environment

- Install Istio
- Deploy httpbin:
```bash
kubectl label namespace default istio-injection=enabled
kubectl create -f https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml
```
- Expose httpbin with Gateway and VirtualService:
```bash
kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: gateway
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - '*.local.kyma.dev'
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: httpbin
spec:
  hosts:
    - 'httpbin.local.kyma.dev'
  gateways:
    - gateway
  http:
    - route:
        - destination:
            host: httpbin.default.svc.cluster.local
            port:
              number: 8000
EOF
```

- Additionally, we can enable rate limit metrics for httpbin:
```bash
kubectl patch deployment httpbin --type merge -p '{"spec":{"template":{"metadata":{"annotations":{"proxy.istio.io/config":"proxyStatsMatcher:\n  inclusionRegexps:\n  - \".*http_local_rate_limit.*\""}}}}}'
```

### Scenario 1: Rate limiting by header value

Rate limit requests based on the value of a header where the rate limiting is defined for each
header value and a fallback for unknown values .

- Apply configuration
```bash
kubectl apply -f ./local/scenario-1-limit-by-static-header-values.yaml
```
- Test rate limiting by sending requests with PRO and BASIC values in the header
```bash
# BASIC allows 2 requests every 20s
while true; do curl -H "x-api-usage:BASIC" -X GET "http://httpbin.local.kyma.dev/get"; done
# PRO allows 100 requests and refills 50 request tokens every 20s
while true; do curl -H "x-api-usage:PRO" -X GET "http://httpbin.local.kyma.dev/get"; done
# All other are limited only by the global token bucket and therefore have no specific restriction
while true; do curl -H "x-api-usage:TEST" -X GET "http://httpbin.local.kyma.dev/get"; done
```

#### Scenario 2: Rate limit by header existence
Rate limit requests based on the existence of a header. The requests are rate limited based on the existence of the `x-limit` header.

- Apply configuration
```bash
kubectl apply -f ./local/scenario-2-limit-by-header-existence.yaml
```
- Test rate limiting by sending requests with x-limit header and without
```bash
# Allows 2 requests every 20s
curl -ik -H "x-limit:true" -X GET "http://httpbin.local.kyma.dev/get"
# Limited by the global token bucket that is restricted to 1000 requests every second
curl -ik -X GET "http://httpbin.local.kyma.dev/get"
```

#### Scenario 3: Rate limiting by client IP

For local rate limiting it's only possible to rate limit by the client IP if the client IP is configured in the rate limiting configuration. It is mentioned in [an issue on GitHub](https://github.com/envoyproxy/envoy/issues/21734#issuecomment-1162667542), that 
using `remote_address` without an IP might be not supported for a local rate limit filter.

- Configure `spec.config.gatewayExternalTrafficPolicy: Local` in Istio CR to preserve the client IP
```bash
kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy":"Local"}}}'
```
- Update IP for the remote_address descriptor to match the local IP used for the calls and apply the configuration:
```bash
kubectl apply -f ./local/scenario-3-limit-by-client-ip.yaml
```
- Test rate limiting, 2 requests every 20s are allowed
```bash
curl -ik -X GET "http://httpbin.local.kyma.dev/get"
```

#### Scenario 4: Rate limiting by client cert
There is no out of the box [Envoy RateLimit Action](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route_components.proto#config-route-v3-ratelimit-action) that supports the `X-Forwarded-Client-Cert` header. In this example the `request_headers` is used to extract the value from `X-Forwarded-Client-Cert` header.
If we want to match the client cert only partial the `header_value_match` RateLimitAction can be used.

- Update `CLIENT_CERT` descriptor to match one of the `X-Forwarded-Client-Cert` values added by ingressgateway and apply the configuration:
```bash
kubectl apply -f ./local/scenario-4-limit-by-static-client-cert.yaml
```
- Test rate limiting by using `X-Forwarded-Client-Cert` added by Ingress Gateway. Since rate limiting is applied on sidecar
```bash
curl -i -X GET "http://httpbin.local.kyma.dev/get"
```

#### Scenario 5: Rate limiting by multiple criteria: custom header existence and IP
- Configure `spec.config.gatewayExternalTrafficPolicy: Local` in Istio CR to preserve the client IP
```bash
kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy":"Local"}}}'
```
- Update IP for the remote_address descriptor to match the local IP used for the calls and apply the configuration:
```bash
kubectl apply -f ./local/scenario-5-limit-by-multiple-criteria.yaml
```
- Test rate limiting
```bash
# 2 requests every 20s are allowed when x-limit header is present
curl -ik -H "x-limit:true" -X GET "http://httpbin.local.kyma.dev/get"
# Default token bucket is used, when request does not contain x-limit header
curl -ik -X GET "http://httpbin.local.kyma.dev/get"
```

### Scenario 6: Rate limiting for all requests, even if no other descriptor is matching
This is handled by the default bucket on the filter level.

### Scenario 7: Rate limiting on Gateway
- Apply configuration
```bash
kubectl apply -f ./local/scenario-7-limit-on-gateway.yaml
```
- Test rate limiting
```bash
# 2 requests every 20s are allowed when x-limit header is present. Since there might be multiple ingress gateways running and requests are distributed requests might not be rate limited after the first 2 requests.
curl -ik -H "x-limit:true" -X GET "http://httpbin.local.kyma.dev/get"
```

## Performance test results
