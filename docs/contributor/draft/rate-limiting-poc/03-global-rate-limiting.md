# Global Rate Limiting
A custom rate limiting service must implement [Envoy's rate limit service protocol](https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/ratelimit/v3/rls.proto).
In the following examples we will use the reference [rate limit service implementation provided by Envoy](https://github.com/envoyproxy/ratelimit).
The reference rate limit service implementation can load multiple rate limit configurations based on a domain. The domain in this case is just a name to scope the rate limit configuration.

## Rate Limit Enforcement
The rate limit service implementation supports a [Shadow mode](https://github.com/envoyproxy/ratelimit?tab=readme-ov-file#shadowmode) where rate limit decisions are tracked without enforcing them.
This shadow mode can be configured [globally](https://github.com/envoyproxy/ratelimit?tab=readme-ov-file#global-shadowmode) or [per descriptor](https://github.com/envoyproxy/ratelimit?tab=readme-ov-file#descriptor-list-definition).

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
- Deploy default rate-limit config and rate limiting service and Redis:
```bash
kubectl create ns ratelimit

kubectl apply -n ratelimit -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ratelimit-config
data:
  config.yaml: |
    domain: ratelimit
    descriptors:
      - key: PATH
        rate_limit:
          unit: minute
          requests_per_unit: 100
EOF

kubectl apply -n ratelimit -f https://raw.githubusercontent.com/istio/istio/release-1.21/samples/ratelimit/rate-limit-service.yaml
```
- Forward Redis port to localhost:
```bash
kubectl port-forward -n ratelimit svc/redis 6379:6379
```
- Additionally, we can enable rate limit metrics for httpbin:
```bash
kubectl patch deployment httpbin --type merge -p '{"spec":{"template":{"metadata":{"annotations":{"proxy.istio.io/config":"proxyStatsMatcher:\n  inclusionRegexps:\n  - \".*http_local_rate_limit.*\""}}}}}'
```

### Scenario 1: Rate limiting by header values

#### Config for rate limiting on static header values
Rate limit requests based on the value of a header where the rate limiting is defined for each 
header value and a fallback for unknown values .

- Apply configuration
```bash
kubectl apply -f ./global/scenario-1-limit-by-static-header-values.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```

- Test
```bash
# BASIC allows 5 requests
while true; do curl -H "x-api-usage:BASIC" -X GET "http://httpbin.local.kyma.dev/get"; done
# PRO allows 10 requests
while true; do curl -H "x-api-usage:PRO" -X GET "http://httpbin.local.kyma.dev/get"; done
# All other allow 1 request
while true; do curl -H "x-api-usage:TEST" -X GET "http://httpbin.local.kyma.dev/get"; done
```

#### Config for rate limiting on dynamic header values
Rate limit requests based on the value of a header where each header value will have its own rate limit budget.

- Apply configuration
```bash
kubectl apply -f ./global/scenario-1-limit-by-dynamic-header-values.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```
- Test rate limiting, each user-id has its own request limit
```bash
curl -i -H "x-user-id:1" -X GET "http://httpbin.local.kyma.dev/get"
curl -i -H "x-user-id:2" -X GET "http://httpbin.local.kyma.dev/get"
curl -i -H "x-user-id:3" -X GET "http://httpbin.local.kyma.dev/get"
```

### Scenario 2: Rate limit by header existence
Rate limit requests based on the existence of a header. In this example, we will rate limit requests based on the `x-limit` header.

- Apply configuration
```bash
kubectl apply -f ./global/scenario-2-limit-by-header-existence.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```
- Test rate limiting, requests with the `x-limit` header are limited to 5 requests per minute
```bash
curl -i -H "x-limit:true" -X GET "http://httpbin.local.kyma.dev/get"
curl -i -X GET "http://httpbin.local.kyma.dev/get"
```

### Scenario 3: Rate limiting by client IP

- Configure `spec.config.gatewayExternalTrafficPolicy: Local` in Istio CR to preserve the client IP
```bash
kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy":"Local"}}}'
```
- Apply configuration
```bash
kubectl apply -f ./global/scenario-3-limit-by-client-ip.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```
- Test rate limit with different IPs is possible by setting numTrustedProxies to 1 and add a XFF header to the requests.
```bash
kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"numTrustedProxies":1}}}'
```
- Send a fake client IP in the XFF header, each IP has its own rate limit of 5 requests per minute
```bash
curl -i -H "X-Forwarded-For:172.0.10.9" -X GET "http://httpbin.local.kyma.dev/get"
curl -i -H "X-Forwarded-For:175.1.23.5" -X GET "http://httpbin.local.kyma.dev/get"
```

### Scenario 4: Rate limiting by client cert

- Apply configuration
```bash
kubectl apply -f ./global/scenario-4-limit-by-client-cert.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```

- Test rate limiting
```bash
curl -i -X GET "http://httpbin.local.kyma.dev/get"
curl --cert client.crt --key client.key --cacert cacert.crt -X GET http://httpbin.local.kyma.dev/get
```

### Scenario 5: Rate limiting by multiple criteria: Rate Limit on header value and IP or existence of different header

- Configure `spec.config.gatewayExternalTrafficPolicy: Local` in Istio CR to preserve the client IP
```bash
kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy":"Local"}}}'
```
- Apply configuration
```bash
kubectl apply -f ./global/scenario-5-limit-by-multiple-criteria.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```
- Test rate limiting, the descriptors for rate limiting is "client IP + x-user-id header" and "x-limit header"
```bash
curl -i -H "X-Forwarded-For:172.0.10.9" -H "x-user-id:1" -X GET "http://httpbin.local.kyma.dev/get"
curl -i -H "X-Forwarded-For:172.0.10.9" -H "x-user-id:2" -X GET "http://httpbin.local.kyma.dev/get"

curl -i -H "X-Forwarded-For:175.1.23.5" -H "x-user-id:1" -X GET "http://httpbin.local.kyma.dev/get"
curl -i -H "X-Forwarded-For:175.1.23.5" -H "x-user-id:2" -X GET "http://httpbin.local.kyma.dev/get"

curl -i -H "x-limit:true" -X GET "http://httpbin.local.kyma.dev/get"
```

### Scenario 6: Rate limiting for all requests, even if no other descriptor is matching
- Configure `spec.config.gatewayExternalTrafficPolicy: Local` in Istio CR to preserve the client IP
```bash
kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy":"Local"}}}'
```
- Apply configuration
```bash
kubectl apply -f ./global/scenario-6-limit-for-all-requests.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```
- Test rate limiting
```bash
# Requests with x-user-id 1 header have a rate limit of 5 requests per minute
curl -i -H "X-Forwarded-For:175.1.23.5" -H "x-user-id:1" -X GET "http://httpbin.local.kyma.dev/get"

# Requests with x-user-id 2 header have a rate limit of 5 requests per minute
curl -i -H "X-Forwarded-For:175.1.23.5" -H "x-user-id:2" -X GET "http://httpbin.local.kyma.dev/get"

# Requests without x-user-id header have a rate limit of 100 requests per minute - requests with x-user-id headers
curl -i -H "X-Forwarded-For:175.1.23.5" -X GET "http://httpbin.local.kyma.dev/get"
```

### Scenario 7: Rate limiting on Service Sidecar
- Configure `spec.config.gatewayExternalTrafficPolicy: Local` in Istio CR to preserve the client IP
```bash
kubectl patch istios -n kyma-system default --type merge -p '{"spec":{"config":{"gatewayExternalTrafficPolicy":"Local"}}}'
```
- Apply configuration
```bash
kubectl apply -f ./global/scenario-7-limit-on-service-sidecar.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```
- Test rate limiting, each user-id has its own request limit of 5 requests per minute
```bash
curl -i -H "x-user-id:1" -X GET "http://httpbin.local.kyma.dev/get"

curl -i -H "x-user-id:2" -X GET "http://httpbin.local.kyma.dev/get"
```

## Performance test results
