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
