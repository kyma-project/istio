# Configuring envoy rate limiting with memcached

1. Apply the following resources:

```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: memcached
  namespace: test
  labels:
    app: memcached
spec:
  ports:
  - name: memcached
    port: 11211
  selector:
    app: memcached
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: memcached
  namespace: test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: memcached
  template:
    metadata:
      labels:
        app: memcached
    spec:
      containers:
      - image: memcached:alpine
        imagePullPolicy: Always
        name: memcached
        ports:
        - name: memcached
          containerPort: 11211
      restartPolicy: Always
      serviceAccountName: ""
---
apiVersion: v1
kind: Service
metadata:
  name: ratelimit
  namespace: test
  labels:
    app: ratelimit
spec:
  ports:
  - name: http-port
    port: 8080
    targetPort: 8080
    protocol: TCP
  - name: grpc-port
    port: 8081
    targetPort: 8081
    protocol: TCP
  - name: http-debug
    port: 6070
    targetPort: 6070
    protocol: TCP
  selector:
    app: ratelimit
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ratelimit
  namespace: test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ratelimit
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: ratelimit
    spec:
      containers:
      - image: envoyproxy/ratelimit:9d8d70a8 # 2022/08/16
        imagePullPolicy: Always
        name: ratelimit
        command: ["/bin/ratelimit"]
        env:
        - name: LOG_LEVEL
          value: debug
        - name: MEMCACHE_HOST_PORT
          value: memcached:11211
        - name: BACKEND_TYPE
          value: memcache
        - name: CACHE_KEY_PREFIX
          value: ratelimit
        - name: USE_STATSD
          value: "false"
        - name: RUNTIME_ROOT
          value: /data
        - name: RUNTIME_SUBDIRECTORY
          value: ratelimit
        - name: RUNTIME_WATCH_ROOT
          value: "false"
        - name: RUNTIME_IGNOREDOTFILES
          value: "true"
        - name: HOST
          value: "::"
        - name: GRPC_HOST
          value: "::"
        ports:
        - containerPort: 8080
        - containerPort: 8081
        - containerPort: 6070
        volumeMounts:
        - name: config-volume
          mountPath: /data/ratelimit/config
      volumes:
      - name: config-volume
        configMap:
          name: ratelimit-config
EOF
```

2. Configure your custom rate limiting config:

```
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: ratelimit-config
  namespace: test
data:
  config.yaml: |
    domain: ratelimit
    descriptors:
      - key: remote_address
        rate_limit:
          unit: minute
          requests_per_unit: 2
EOF
```

3. Apply envoy filters pointing to above ratelimit service:

```
kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: rate-limit-actions
  namespace: istio-system
spec:
  workloadSelector:
    labels:
      istio: ingressgateway
  configPatches:
    - applyTo: VIRTUAL_HOST
      match:
        context: GATEWAY
        routeConfiguration:
          vhost:
            name: ""
            route:
              action: ANY
      patch:
        operation: MERGE
        value:
          rate_limits:
            - actions:
                - remote_address: { }
EOF
```

```
kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: global-ratelimit-service
  namespace: istio-system
spec:
  workloadSelector:
    labels:
      istio: ingressgateway
  configPatches:
    - applyTo: HTTP_FILTER
      match:
        context: GATEWAY
        listener:
          filterChain:
            filter:
              name: "envoy.filters.network.http_connection_manager"
              subFilter:
                name: "envoy.filters.http.router"
      patch:
        operation: INSERT_BEFORE
        value:
          name: envoy.filters.http.ratelimit
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit
            stat_prefix: global_rate_limiter
            enable_x_ratelimit_headers: DRAFT_VERSION_03
            # domain can be anything! Match it to the ratelimter service config
            domain: ratelimit
            failure_mode_deny: true
            timeout: 10s
            rate_limit_service:
              grpc_service:
                envoy_grpc:
                  cluster_name: outbound|8081||ratelimit.test.svc.cluster.local
                  authority: ratelimit.test.svc.cluster.local
              transport_api_version: V3
EOF
```

# Test scenarios for memcached rate limiting

### Scenario 1: Rate limiting by header values

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
curl -i -H "x-user-id:1" -X GET "http://httpbin.ps-rate.goatz.shoot.canary.k8s-hana.ondemand.com/get"
curl -i -H "x-user-id:2" -X GET "http://httpbin.ps-rate.goatz.shoot.canary.k8s-hana.ondemand.com/get"
curl -i -H "x-user-id:3" -X GET "http://httpbin.ps-rate.goatz.shoot.canary.k8s-hana.ondemand.com/get"
curl -i -H "x-user-id:50e29ebe23f0716b5bfbe42ac9c4c8e75f46899c79e31a76fc11fe62e1e6f3a4770272a6d1119f8526ee668ddb1c28b70e9542f638c8251bc7802fdd205b7614356940285ec6da05af3e1eda659c72cb4df3367a7a261c850fee2e85176c39161ac86c93109c1fc8648c524cb8af745f7dfcbff448b2e49721195b6262a4326450e29ebe23f0716b5bfbe42ac9c4c8e75f46899c79e31a76fc11fe62e1e6f3a4770272a6d1119f8526ee668ddb1c28b70e9542f638c8251bc7802fdd205b7614356940285ec6da05af3e1eda659c72cb4df3367a7a261c850fee2e85176c39161ac86c93109c1fc8648c524cb8af745f7dfcbff448b2e49721195b6262a43264" -X GET "http://httpbin.ps-rate.goatz.shoot.canary.k8s-hana.ondemand.com/get"
```

Getting error with the long one:
```
time="2024-06-13T12:57:20Z" level=error msg="Error multi-getting memcache keys ([ratelimitratelimit_USER_ID_50e29ebe23f0716b5bfbe42ac9c4c8e75f46899c79e31a76fc11fe62e1e6f3a4770272a6d1119f8526ee668ddb1c28b70e9542f638c8251bc7802fdd205b7614356940285ec6da05af3e1eda659c72cb4df3367a7a261c850fee2e85176c39161ac86c93109c1fc8648c524cb8af745f7dfcbff448b2e49721195b6262a4326450e29ebe23f0716b5bfbe42ac9c4c8e75f46899c79e31a76fc11fe62e1e6f3a4770272a6d1119f8526ee668ddb1c28b70e9542f638c8251bc7802fdd205b7614356940285ec6da05af3e1eda659c72cb4df3367a7a261c850fee2e85176c39161ac86c93109c1fc8648c524cb8af745f7dfcbff448b2e49721195b6262a43264_1718283420]): malformed: key is too long or contains invalid characters"
```

### Scenario 4: Rate limiting by client cert
There is no out of the box [Envoy RateLimit Action](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route_components.proto#config-route-v3-ratelimit-action) that supports the `X-Forwarded-Client-Cert` header. In this example the `request_headers` is used to extract the value from `X-Forwarded-Client-Cert` header.

- Apply configuration
```bash
kubectl apply -f ./global/scenario-4-limit-by-client-cert.yaml
```
- Restart rate limit service to apply the new config
```bash
kubectl delete pod -n ratelimit -l app=ratelimit
```
- Test rate limiting by using `X-Forwarded-Client-Cert` added by Ingress Gateway. Since rate limiting is applied on sidecar
```bash
curl -i -X GET "http://httpbin.ps-rate.goatz.shoot.canary.k8s-hana.ondemand.com/get"
```

Client cert is not working due to too long key value:
```
time="2024-06-13T13:05:28Z" level=error msg="Error multi-getting memcache keys ([ratelimitratelimit_CLIENT_CERT_By=spiffe://cluster.local/ns/default/sa/httpbin;Hash=8e5bed411f57a6f82015aea05234c6b8bba2ce8efb73b2cc0965ab9660213d98;Subject=\"\";URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account_1718283900]): malformed: key is too long or contains invalid characters"
```
