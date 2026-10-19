# Diagnosing 404 Errors
<!-- open-source-only -->

Follow this troubleshooting guide if you receive the `404 Not Found` status code when trying to access a Kyma endpoint. A `404` can come from incomplete routing in the mesh, from a Gateway conflict, or from your application, and the fix differs in each case.

## Identify the Origin of the `404` Error

A `404` error from incomplete routing, a `404` error from a Gateway conflict, and a `404` error from your application look similar, but you can tell them apart by inspecting the response and the mesh configuration.

The decisive signals are the `server` header, the `x-envoy-upstream-service-time` header, and the response body. The `x-envoy-upstream-service-time` header is present only when Envoy forwarded the request to the upstream workload and measured its response time. When no route matches, Envoy returns the `404` error itself, so this header is absent. Your application responds only after the request reaches it, so this header is present.

Send a request to the failing endpoint and include the response headers:

```bash
curl -sD - http://{YOUR_HOST}/{YOUR_PATH}
```

Compare your output with the following cases:

| Aspect | Incomplete Routing | Gateway Conflict | Application |
|---|---|---|---|
| Where it runs | In the Envoy proxy, which finds no matching route | In Istio Ingress Gateway, which cannot bind the host | In your workload, behind the Envoy proxy |
| `server` response header | `istio-envoy` | `istio-envoy` | `istio-envoy` |
| `x-envoy-upstream-service-time` header | Absent | Absent | Present |
| Response body | Empty | Empty | Your application's own payload |
| `istioctl analyze` | No conflict reported | Reports an `IST0145` conflict | No conflict reported |
| Fixed by | Adding or correcting a VirtualService route | Making the host unique across Gateway resources | Debugging the application |

To decide which case applies, use the following signals:

- The response has the `server: istio-envoy` header, an empty body, and no `x-envoy-upstream-service-time` header. The request reached the mesh, but no route matched it. Run `istioctl analyze` to check for a Gateway conflict. If `istioctl analyze` reports no conflict, the routing configuration is incomplete. See [The Routing Configuration Is Incomplete](#the-routing-configuration-is-incomplete). If it reports an `IST0145` conflict, two Gateway resources share the same host. See [Two Gateway Resources Share the Same Host](#two-gateway-resources-share-the-same-host).
- The response has the `server: istio-envoy` header, your application's own body, and the `x-envoy-upstream-service-time` header. Your application generated the `404` error. See [The `404` Comes from Your Application](#the-404-comes-from-your-application).

> ### NOTE:
> Use the response body and the presence of the `x-envoy-upstream-service-time` header to decide the origin, not the `server` header. The proxy adds `server: istio-envoy` to all three responses.

## The Routing Configuration Is Incomplete

### Symptom

The request reaches the mesh but no VirtualService route matches the requested host and path, so the Envoy proxy returns an empty `404` response with the `server: istio-envoy` header and no `x-envoy-upstream-service-time` header:

```bash
curl -sD - http://{YOUR_HOST}/nope
```
```
HTTP/1.1 404 Not Found
date: Tue, 20 Oct 2026 14:42:13 GMT
server: istio-envoy
content-length: 0
```

### Cause

The Gateway resource exposes the host, but no VirtualService defines a route for the requested path. For example, a VirtualService that matches only the `/app` prefix does not match a request to `/nope`.

### Solution

To find out which routes the Envoy proxy applies, inspect the live route configuration of Istio Ingress Gateway.

1. Get the name of the Istio Ingress Gateway Pod:

    ```bash
    kubectl get pods --namespace istio-system --selector istio=ingressgateway --output name
    ```

2. List the routes that the Istio Ingress Gateway Pod applies:

    ```bash
    istioctl proxy-config routes {POD_NAME}.istio-system
    ```
    ```
    NAME          VHOST NAME                 DOMAINS                 MATCH                  VIRTUAL SERVICE
    http.8080     app-404.example.com:80     app-404.example.com     /app                   app-404.default
    ```

    In this example, the proxy routes only the `/app` path. A request to any other path receives a `404` error.

3. List the VirtualService resources and check their host, Gateway, and routes:

    ```bash
    kubectl get virtualservices --all-namespaces
    ```

Verify that a VirtualService exposes the requested host and path, and that it binds to the correct Gateway resource. If no route matches, add or correct the VirtualService route. For more information about debugging Istio traffic resources, see [Network Connectivity - Diagnostics](./03-01-network-connectivity.md#debugging-istio-resources).

## Two Gateway Resources Share the Same Host

### Symptom

Two Gateway resources define the same host, so Istio Ingress Gateway cannot reliably bind the host to a single Gateway resource, and requests receive an empty `404` response with the `server: istio-envoy` header and no `x-envoy-upstream-service-time` header. Running `istioctl analyze` reports an `IST0145` conflict.

### Cause

When you create two Gateway resources with the same host, selector, and port, they conflict. Istio Ingress Gateway cannot match an incoming request to a specific Gateway resource. For more information about this behavior, see the [Istio documentation](https://istio.io/latest/docs/ops/common-problems/network-issues/#404-errors-occur-when-multiple-gateways-configured-with-same-tls-certificate).

A conflict can span namespaces. For example, a Gateway resource in the `default` namespace and a Gateway resource in another namespace can both claim the same host.

When you create Ingress resources that use Istio as their ingress class, a Gateway entry is also created underneath. To check the cluster's configuration, run the `istioctl x internal-debug configz` command.

### Solution

To detect a Gateway conflict, run `istioctl analyze`. The analyzer reports an `IST0145` error for each Gateway resource that shares the host:

```bash
istioctl analyze --all-namespaces
```
```
Error [IST0145] (Gateway default/app-404-gateway) Conflict with gateways team-b/other-gateway (workload selector istio=ingressgateway, port 80, hosts app-404.example.com).
Error [IST0145] (Gateway team-b/other-gateway) Conflict with gateways default/app-404-gateway (workload selector istio=ingressgateway, port 80, hosts app-404.example.com).
```

To resolve the conflict, verify that a host matches only one Gateway resource. Remove the duplicate host from all but one Gateway resource, or assign a unique host to each Gateway resource.

## The `404` Comes from Your Application

### Symptom

The request reaches your workload and the application returns the `404` error. The proxy still adds the `server: istio-envoy` header, but the response also carries the `x-envoy-upstream-service-time` header and your application's own body:

```bash
curl -sD - http://{YOUR_HOST}/missing
```
```
HTTP/1.1 404 Not Found
server: istio-envoy
date: Tue, 20 Oct 2026 14:36:38 GMT
content-type: application/octet-stream
content-length: 29
x-envoy-upstream-service-time: 4

not found by the application
```

### Cause

Istio routes the request correctly, but the application rejects it with its own `404` response, for example because the requested path does not exist in the application. Because the response originates in the workload, no Istio configuration change fixes it.

### Solution

To confirm that the application returns the `404` error, bypass the mesh and send the request directly to the Pod. If the Pod still returns the `404` error, the application is the source, not Istio.

1. Forward a local port to the workload Pod:

    ```bash
    kubectl port-forward deployment/{DEPLOYMENT_NAME} 18080:{CONTAINER_PORT} --namespace {NAMESPACE}
    ```

2. In another terminal, send the request to the forwarded port and include the response headers:

    ```bash
    curl -sD - http://127.0.0.1:18080/{YOUR_PATH}
    ```
    ```
    HTTP/1.1 404 Not Found
    Server: nginx/1.30.6

    not found by the application
    ```

    The `Server` header now shows the application server, such as `nginx`, instead of `istio-envoy`, because the request did not pass through the proxy. The `404` response still appears, which confirms that the application generates it.

Because the application returns the `404` error, debug the workload, not Istio. Check the application's routing and configuration, and verify that the requested path exists.
