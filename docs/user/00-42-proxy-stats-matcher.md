# Proxy Stats Matcher

Learn how to configure `proxyStatsMatcher` to emit additional statistics from Istio sidecar and gateway proxies.

## Overview

Istio proxies always emit a built-in default set of statistics (such as `cluster_manager`, `listener_manager`, `server`, and `cluster.xds-grpc`). Many other statistic families are suppressed by default to reduce CPU and memory overhead. The `proxyStatsMatcher` setting lets you opt in to additional statistics by matching their names against regular expressions.

Use this setting when you want to:

- Enable a specific group of additional proxy statistics for troubleshooting or observability
- Expose statistics for a specific feature, such as outlier detection
- Keep the configuration focused on the metrics you actually need

## How `proxyStatsMatcher` Works

The recommended way to configure `proxyStatsMatcher` is through the `spec.config` section of the `Istio` CR. This applies the configuration globally to all proxies in the mesh.

For cases where you need additional statistics only on specific workloads — for example, when troubleshooting a single service — you can use the `proxy.istio.io/config` annotation on the Pod template instead. Be aware that when the annotation is set, it **replaces** the global `proxyStatsMatcher` for that workload rather than extending it. If you want a workload to include both the globally configured patterns and additional ones, you must repeat the global patterns in the annotation.

The `inclusionRegexps` field contains a list of regular expressions. Statistics whose names match at least one expression are emitted in addition to the built-in default set.

### Global Mesh Configuration

To enable additional statistics for all managed proxies, configure the `Istio` CR:
<!-- tabs:start -->
#### **Kyma Dashboard**

1. Go to **Cluster Details** in Kyma dashboard.
2. In the `kyma-system` namespace, go to the **Istio** section.
3. Choose **Edit**.
4. Under **Proxy Stats Matcher**, add an entry to **Inclusion Regexps**.
5. Save the changes.

#### **Using kubectl**

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    proxyStatsMatcher:
      inclusionRegexps:
        - ".*outbound.*"
```

You can also use `kubectl patch`:

```bash
kubectl patch istio default -n kyma-system --type=merge -p '{"spec":{"config":{"proxyStatsMatcher":{"inclusionRegexps":[".*outbound.*"]}}}}'
```
<!-- tabs:end -->

To verify the configuration, run:

```bash
kubectl get istio default -n kyma-system -o jsonpath='{.spec.config.proxyStatsMatcher.inclusionRegexps}'
```

### Per-Workload Configuration

If you need additional statistics only on a specific workload, use the `proxy.istio.io/config` annotation on the Pod template instead of the global CR setting:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        proxy.istio.io/config: |-
          proxyStatsMatcher:
            inclusionRegexps:
              - ".*outbound.*"
    spec:
      containers:
        - name: my-app
          image: my-app:latest
```

Restart the affected workloads after changing the annotation so that the updated sidecar configuration is applied.

### Configure Multiple Matchers

You can provide more than one regular expression to emit statistics for several areas of interest:

```yaml
spec:
  config:
    proxyStatsMatcher:
      inclusionRegexps:
        - ".*outbound.*"
        - ".*upstream_rq_retry.*"
```

Choose the narrowest expressions that satisfy your use case to avoid enabling unnecessary additional statistics.

## Verify the Configuration

To inspect matching statistics for a specific workload, run the following command in the `istio-proxy` container:

```bash
kubectl exec -n <namespace> <pod-name> -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outbound
```

Replace `<namespace>` and `<pod-name>` with the namespace and name of a Pod that has an Istio sidecar proxy.

## Tutorial: Enable Outlier Detection Metrics

This tutorial shows how to set up a complete scenario in which outlier detection is configured and its metrics are observed via `proxyStatsMatcher`. You deploy two namespaces — a `target` namespace with an HTTP workload and a `source` namespace with sidecar-injected curl pods — configure a `DestinationRule` with outlier detection, trigger ejections, and observe the resulting metrics.

### Prerequisites

- A running Kubernetes cluster with Kyma Istio installed.
- `kubectl` configured to access the cluster.

### Context

Outlier detection is configured in a `DestinationRule` on the target host but is **evaluated by the source proxy** that makes requests. As a result, each source proxy tracks ejection state independently — one proxy may eject a host while another considers it healthy. This is why metrics must be observed on the source proxy, not the target.

When you configure `proxyStatsMatcher` globally via the `Istio` CR, **all proxies in the mesh** — including ingress and egress gateways — receive that matcher configuration. Matching statistics can then be emitted by those proxies, while host-specific metrics appear when a given proxy has relevant traffic or state for that destination.

### Steps

#### 1. Enable Outlier Detection Metrics Globally

Configure the Istio CR to emit outlier detection statistics on all proxies in the mesh:

```bash
kubectl patch istio default -n kyma-system --type=merge -p '{"spec":{"config":{"proxyStatsMatcher":{"inclusionRegexps":[".*outlier_detection.*"]}}}}'
```

Verify the configuration:

```bash
kubectl get istio default -n kyma-system -o jsonpath='{.spec.config.proxyStatsMatcher.inclusionRegexps}'
```

All proxies in the mesh — sidecars and gateways — will now emit outlier detection metrics.

#### 2. Prepare Namespaces

Create the target and source namespaces. Enable Istio sidecar injection only on the source:

```bash
kubectl create ns target
kubectl create ns source
kubectl label namespace source istio-injection=enabled
```

#### 3. Deploy the Target Workload

Deploy an `httpbin` workload in the `target` namespace:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: httpbin
  namespace: target
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
  namespace: target
  labels:
    app: httpbin
spec:
  ports:
    - name: http
      port: 8000
      targetPort: 80
  selector:
    app: httpbin
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  namespace: target
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
  template:
    metadata:
      labels:
        app: httpbin
    spec:
      serviceAccountName: httpbin
      containers:
        - image: docker.io/kennethreitz/httpbin
          imagePullPolicy: IfNotPresent
          name: httpbin
          ports:
            - containerPort: 80
```

#### 4. Configure Outlier Detection

Apply a `DestinationRule` that ejects the host after five consecutive 5xx errors within a one-minute interval:

```yaml
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: httpbin-policy
  namespace: target
spec:
  host: httpbin.target.svc.cluster.local
  trafficPolicy:
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 1m
      baseEjectionTime: 5m
      maxEjectionPercent: 100
```

#### 5. Deploy Source Pods

Deploy two curl pods in the `source` namespace. Since `proxyStatsMatcher` is configured globally, metrics are automatically enabled on all proxies:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: curl-1
  namespace: source
  labels:
    app: curl-1
spec:
  containers:
    - name: curl
      image: curlimages/curl
      command: ["/bin/sleep", "36000"]
---
apiVersion: v1
kind: Pod
metadata:
  name: curl-2
  namespace: source
  labels:
    app: curl-2
spec:
  containers:
    - name: curl
      image: curlimages/curl
      command: ["/bin/sleep", "36000"]
```

Wait for both pods to be ready:

```bash
kubectl wait --for=condition=ready pod -n source curl-1 curl-2
```

#### 6. Trigger Host Ejection

Send five consecutive 5xx responses from `curl-1` to exceed the outlier detection threshold:

```bash
for status in 501 502 503 504 505; do
  kubectl exec -n source curl-1 -c curl -- curl -s -o /dev/null "http://httpbin.target.svc.cluster.local:8000/status/${status}"
done
```

After the threshold is reached, the host is ejected from the load-balancing pool as seen from `curl-1`.

#### 7. Observe the Ejection

Send one more request from `curl-1`. It fails because the host is ejected:

```bash
kubectl exec -n source curl-1 -c curl -- curl -v "http://httpbin.target.svc.cluster.local:8000/headers"
```

Send the same request from `curl-2`. It succeeds because `curl-2` has its own independent ejection state:

```bash
kubectl exec -n source curl-2 -c curl -- curl -v "http://httpbin.target.svc.cluster.local:8000/headers"
```

#### 8. Inspect Outlier Detection Metrics

Check the metrics on `curl-1`. The active ejection count should be `1`:

```bash
kubectl exec -n source curl-1 -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
```

Expected output:

```
# TYPE envoy_cluster_outlier_detection_ejections_active gauge
envoy_cluster_outlier_detection_ejections_active{cluster_name="outbound|8000||httpbin.target.svc.cluster.local"} 1
```

Check the metrics on `curl-2`. The active ejection count should be `0` because no ejection was triggered from this pod:

```bash
kubectl exec -n source curl-2 -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
```

Expected output:

```
# TYPE envoy_cluster_outlier_detection_ejections_active gauge
envoy_cluster_outlier_detection_ejections_active{cluster_name="outbound|8000||httpbin.target.svc.cluster.local"} 0
```

#### 9. (Optional) Test Outlier Detection via Ingress Gateway

When `proxyStatsMatcher` is configured globally, it propagates to the ingress gateway as well as sidecars. The gateway proxy evaluates outlier detection independently — ejection state is local to the proxy that originates the request, so the gateway tracks it separately from any sidecar.

Two key properties follow from this:

- **Ejection ownership**: when requests travel through the ingress gateway to a backend, it is the gateway proxy that records the ejection — not the client that sent traffic to the gateway.
- **Per-replica independence**: when the gateway is scaled to multiple replicas, each pod maintains its own ejection state. A replica that received enough consecutive errors from a backend ejects it independently of replicas that did not.

1. Create a Gateway and VirtualService to route traffic through the ingress:

```yaml
apiVersion: networking.istio.io/v1
kind: Gateway
metadata:
  name: httpbin-gateway
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway
  servers:
  - hosts:
    - 'httpbin.example.com'
    port:
      name: http
      number: 80
      protocol: HTTP
---
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: httpbin
  namespace: istio-system
spec:
  gateways:
  - istio-system/httpbin-gateway
  hosts:
  - 'httpbin.example.com'
  http:
  - route:
    - destination:
        host: httpbin.<target-namespace>.svc.cluster.local
        port:
          number: 8000
```

2. Send enough consecutive 5xx requests through the gateway to exceed the ejection threshold:

```bash
for i in $(seq 1 5); do
  kubectl exec -n <source-namespace> <curl-pod> -c curl -- curl -s -o /dev/null \
    -H "Host: httpbin.example.com" \
    "http://istio-ingressgateway.istio-system.svc.cluster.local/status/500"
done
```

3. Inspect metrics on the ingress gateway pod. The active ejection count should be `1`:

```bash
ingress_pod=$(kubectl get pod -n istio-system -l app=istio-ingressgateway -o jsonpath="{.items[0].metadata.name}")
kubectl exec -n istio-system "${ingress_pod}" -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
```

4. Verify that the curl pod's sidecar reports no active ejection — it sent requests to the gateway, not directly to the backend, so it has no ejection state for that cluster:

```bash
kubectl exec -n <source-namespace> <curl-pod> -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
```

When the gateway is scaled to multiple replicas, only the replicas that processed enough consecutive 5xx responses from the backend eject it. Each replica's `ejections_active` value is consistent with its own upstream 5xx count — a replica that crossed the threshold reports `1`, others report `0`.

## Considerations

- `proxyStatsMatcher` extends the default set of emitted statistics. It does not enable or configure traffic-management features such as outlier detection by itself.
- Broad regular expressions can increase the number of emitted metrics and slightly raise proxy resource usage.
- For outlier detection, ejection state is local to each source proxy. Always inspect metrics on the proxy that generated the traffic.
- After changing the `proxy.istio.io/config` annotation or the global `proxyStatsMatcher` setting, restart the affected workloads for the change to take effect.
- If you use a metrics backend such as Prometheus, make sure that the additionally emitted statistics are also collected and retained according to your observability setup.

## Related Information

- [Istio Custom Resource](./04-00-istio-custom-resource.md)
- [Envoy Statistics](https://istio.io/latest/docs/ops/configuration/telemetry/envoy-stats/)
- [DestinationRule outlier detection](https://istio.io/latest/docs/reference/config/networking/destination-rule/#OutlierDetection)
- [Uneven Traffic Distribution with DestinationRules](./troubleshooting/03-95-uneven-load-balancing-with-destination-rules.md)

