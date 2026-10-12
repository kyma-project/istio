# Enable Outlier Detection Metrics

This tutorial shows how to set up a complete scenario in which outlier detection is configured and its metrics are observed via `proxyStatsMatcher`. You deploy two namespaces (a `target` namespace with an HTTP workload and a `source` namespace with sidecar-injected curl Pods), configure a `DestinationRule` with outlier detection, trigger ejections, and observe the resulting metrics.

## Prerequisites

- A running Kubernetes cluster with the Istio module added.
- `kubectl` configured to access the cluster.

## Context

Outlier detection is configured in a `DestinationRule` on the target host but is **evaluated by the source proxy** that makes requests. As a result, each source proxy tracks ejection state independently — one proxy may eject a host while another considers it healthy. This is why metrics must be observed on the source proxy, not the target.

When you configure `proxyStatsMatcher` globally via the `Istio` CR, **all proxies in the mesh** — including ingress and egress gateways — receive that matcher configuration. As a result, these proxies can emit matching statistics, while host-specific metrics appear when a given proxy has relevant traffic or state for that destination.

## Procedure

1. Enable outlier detection metrics globally.

   Configure the Istio CR to emit outlier detection statistics on all proxies in the mesh:

   ```bash
   kubectl patch istio default -n kyma-system --type=merge -p '{"spec":{"config":{"proxyStatsMatcher":{"inclusionRegexps":[".*outlier_detection.*"]}}}}'
   ```

   Verify the configuration:

   ```bash
   kubectl get istio default -n kyma-system -o jsonpath='{.spec.config.proxyStatsMatcher.inclusionRegexps}'
   ```

   All proxies in the mesh — sidecars and gateways — will now emit outlier detection metrics.

2. Prepare namespaces.

   Create the target and source namespaces. Enable Istio sidecar injection only on the source:

   ```bash
   kubectl create ns target
   kubectl create ns source
   kubectl label namespace source istio-injection=enabled
   ```

3. Deploy the target workload.

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

4. Configure outlier detection.

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

5. Deploy source Pods.

   Deploy two curl Pods in the `source` namespace. Since `proxyStatsMatcher` is configured globally, metrics are automatically enabled on all proxies:

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

   Wait for both Pods to be ready:

   ```bash
   kubectl wait --for=condition=ready pod -n source curl-1 curl-2
   ```

6. Trigger host ejection.

   Send five consecutive 5xx responses from `curl-1` to exceed the outlier detection threshold:

   ```bash
   for status in 501 502 503 504 505; do
     kubectl exec -n source curl-1 -c curl -- curl -s -o /dev/null "http://httpbin.target.svc.cluster.local:8000/status/${status}"
   done
   ```

   After the threshold is reached, the host is ejected from the load-balancing pool as seen from `curl-1`.

7. Observe the ejection.

   Send one more request from `curl-1`. It fails because the host is ejected:

   ```bash
   kubectl exec -n source curl-1 -c curl -- curl -v "http://httpbin.target.svc.cluster.local:8000/headers"
   ```

   Send the same request from `curl-2`. It succeeds because `curl-2` has its own independent ejection state:

   ```bash
   kubectl exec -n source curl-2 -c curl -- curl -v "http://httpbin.target.svc.cluster.local:8000/headers"
   ```

8. Inspect outlier detection metrics.

   Check the metrics on `curl-1`. The active ejection count should be `1`:

   ```bash
   kubectl exec -n source curl-1 -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
   ```

   Expected output:

   ```
   # TYPE envoy_cluster_outlier_detection_ejections_active gauge
   envoy_cluster_outlier_detection_ejections_active{cluster_name="outbound|8000||httpbin.target.svc.cluster.local"} 1
   ```

   Check the metrics on `curl-2`. The active ejection count should be `0` because no ejection was triggered from this Pod:

   ```bash
   kubectl exec -n source curl-2 -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
   ```

   Expected output:

   ```
   # TYPE envoy_cluster_outlier_detection_ejections_active gauge
   envoy_cluster_outlier_detection_ejections_active{cluster_name="outbound|8000||httpbin.target.svc.cluster.local"} 0
   ```

9. (Optional) Test outlier detection via the ingress gateway.

   When `proxyStatsMatcher` is configured globally, it propagates to the ingress gateway and sidecars. The gateway proxy evaluates outlier detection independently — ejection state is local to the proxy that originates the request, so the gateway tracks it separately from any sidecar.

   Two key properties follow from this:

   - **Ejection ownership**: When requests travel through the ingress gateway to a backend, it is the gateway proxy that records the ejection, not the client that sent traffic to the gateway.
   - **Per-replica independence**: When the gateway is scaled to multiple replicas, each Pod maintains its own ejection state. A replica ejects a backend after receiving enough consecutive errors, without affecting replicas that have not reached the error threshold.

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

   3. Inspect metrics on the ingress gateway Pod. The active ejection count should be `1`:

      ```bash
      ingress_pod=$(kubectl get pod -n istio-system -l app=istio-ingressgateway -o jsonpath="{.items[0].metadata.name}")
      kubectl exec -n istio-system "${ingress_pod}" -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
      ```

   4. Verify that the curl Pod's sidecar doesn't report an active ejection. It sent requests to the gateway, not directly to the backend, so it has no ejection state for that cluster:

      ```bash
      kubectl exec -n <source-namespace> <curl-pod> -c istio-proxy -- pilot-agent request GET /stats/prometheus | grep outlier_detection
      ```

## Result

You have configured `proxyStatsMatcher` to emit outlier detection metrics and verified that ejection state is tracked independently per source proxy. The `curl-1` sidecar shows `ejections_active 1`, meaning it has ejected the httpbin backend from its upstream load-balancing pool after receiving consecutive errors. The `curl-2` sidecar shows `ejections_active 0` because it never triggered the ejection threshold - each proxy tracks this state independently.

## Related Information (Optional)

- [Proxy Stats Matcher](../00-42-proxy-stats-matcher.md)
- [Istio Custom Resource](../04-00-istio-custom-resource.md)
- [DestinationRule outlier detection](https://istio.io/latest/docs/reference/config/networking/destination-rule/#OutlierDetection)
