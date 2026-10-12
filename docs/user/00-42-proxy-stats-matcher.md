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
        - "{YOUR_REGEXP}"
```

You can also use `kubectl patch`:

```bash
kubectl patch istio default -n kyma-system --type=merge -p '{"spec":{"config":{"proxyStatsMatcher":{"inclusionRegexps":[".*outlier_detection.*"]}}}}'
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
              - ".*outlier_detection.*"
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
kubectl exec -n <namespace> <pod-name> -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
```

Replace `<namespace>` and `<pod-name>` with the namespace and name of a Pod that has an Istio sidecar proxy.

## Considerations

- `proxyStatsMatcher` extends the default set of emitted statistics. It does not enable or configure traffic-management features such as outlier detection by itself.
- Broad regular expressions can increase the number of emitted metrics and slightly raise proxy resource usage.
- For outlier detection, ejection state is local to each source proxy. Always inspect metrics on the proxy that generated the traffic.
- After changing the `proxy.istio.io/config` annotation or the global `proxyStatsMatcher` setting, restart the affected workloads for the change to take effect.
- If you use a metrics backend such as Prometheus, make sure that the additionally emitted statistics are also collected and retained according to your observability setup.
- When you configure `proxyStatsMatcher` globally via the `Istio` CR, **all proxies in the mesh** — including ingress and egress gateways — receive that matcher configuration. As a result, these proxies can emit matching statistics, while host-specific metrics appear when a given proxy has relevant traffic or state for that destination.
- When the gateway is scaled to multiple replicas, only the replicas that have processed enough consecutive `5xx` responses from the backend eject it. Each replica's `ejections_active` value is consistent with its own upstream `5xx` count — a replica that crossed the threshold reports `1`, others report `0`.

## Related Information

- [Istio Custom Resource](./04-00-istio-custom-resource.md)
- [Envoy Statistics](https://istio.io/latest/docs/ops/configuration/telemetry/envoy-stats/)
- [DestinationRule outlier detection](https://istio.io/latest/docs/reference/config/networking/destination-rule/#OutlierDetection)
- [Uneven Traffic Distribution with DestinationRules](./troubleshooting/03-95-uneven-load-balancing-with-destination-rules.md)
- [Enable Outlier Detection Metrics](./tutorials/01-70-enable-outlier-detection-metrics.md)
