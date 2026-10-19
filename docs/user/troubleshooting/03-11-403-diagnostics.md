# Diagnosing 403 Errors

## Symptom

When you try to access a Kyma endpoint, you get a `403 Forbidden` error.

## Cause

A `403` error can come from three different places, and you fix each one in a different way:

- **Istio returns the `403` error.** An AuthorizationPolicy inside the mesh rejects the request. The Envoy proxy generates the response, and the request never reaches your application.
- **Your application returns the `403` error.** The request is routed correctly and reaches your workload, but the application responds with a `403` on its own, for example because of its own authentication or authorization logic.
- **Something outside the mesh returns the `403` error.** An external firewall, proxy, or web application firewall (WAF) blocks the request before it reaches Envoy.

All three cases can surface as a `403 Forbidden` response, so before you change any Istio configuration, find out which one applies. See [Identify the Origin of the 403](#identify-the-origin-of-the-403).

## Identify the Origin of the 403

A `403` error from an AuthorizationPolicy, a `403` error from your application, and a `403` error from an external firewall look similar, but you can tell them apart by inspecting the response. Identifying the origin saves you from looking in the wrong place.

The decisive signals are the `server` header, the `x-envoy-upstream-service-time` header, and the response body. The `x-envoy-upstream-service-time` header is present only when Envoy forwarded the request to the upstream workload and measured its response time. An AuthorizationPolicy rejects the request before it reaches the workload, so this header is absent. Your application responds only after the request reaches it, so this header is present.

Send a request to the failing endpoint and include the response headers:

```bash
curl -sD - http://{YOUR_HOST}/{YOUR_PATH}
```

Compare your output with the following cases:

| Aspect | AuthorizationPolicy (Istio) | Application | External Firewall, Proxy, or WAF |
|---|---|---|---|
| Where it runs | Inside the mesh, enforced by the Envoy proxy | In your workload, behind the Envoy proxy | Outside the mesh, before the request reaches Envoy |
| `server` response header | `istio-envoy` | `istio-envoy` | Absent, or a non-Envoy value such as a proxy or WAF product name |
| `x-envoy-upstream-service-time` header | Absent | Present | Absent |
| Response body | Plain text `RBAC: access denied` | Your application's own payload | Vendor-specific HTML or JSON error page, or empty |
| Depends on request identity | Yes. The decision depends on the `to`, `from`, and `when` rules, so it varies by path, source, and token | Depends on the application logic | No. The block is usually the same regardless of path, headers, or credentials |
| Fixed by | Updating the AuthorizationPolicy resource | Debugging the application | Cloud provider or network administrator |

To decide which case applies, use the following signals:

- The response has the `server: istio-envoy` header and an `RBAC: access denied` body, and the `x-envoy-upstream-service-time` header is **absent**. An AuthorizationPolicy rejected the request. See [The 403 Comes from an AuthorizationPolicy](#the-403-comes-from-an-authorizationpolicy).
- The response has the `server: istio-envoy` header and your application's own body, and the `x-envoy-upstream-service-time` header is **present**. Your application generated the `403` error. See [The 403 Comes from Your Application](#the-403-comes-from-your-application).
- The response has no `server: istio-envoy` header, or carries a vendor-specific error page instead of `RBAC: access denied`. Something other than Istio produced the `403` error, such as an external proxy or WAF in front of the cluster. This is not an Istio problem. See [The 403 Comes from Outside the Mesh](#the-403-comes-from-outside-the-mesh).

> ### Tip:
> Use the response body and the presence of the `x-envoy-upstream-service-time` header to decide the origin, not the `server` header. The proxy adds `server: istio-envoy` to both an AuthorizationPolicy response and an application response.

## The 403 Comes from an AuthorizationPolicy

If the request reaches the mesh and the Envoy proxy returns the `403` error with the `server: istio-envoy` header and an `RBAC: access denied` body, a misconfigured AuthorizationPolicy rejects the request.

### Cause

A misconfigured AuthorizationPolicy is a common cause of `403` errors. A frequent example is an unnecessary port number in the Host header. Istio checks the host as-is. If the Host header contains a port number and the AuthorizationPolicy defines only a hostname, Istio denies the request.

The following AuthorizationPolicy allows only the hostname `httpbin.local.kyma.dev`, without a port:

```yaml
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: ingress-allow-headers
  namespace: istio-system
spec:
  action: ALLOW
  rules:
  - to:
    - operation:
        hosts: [ "httpbin.local.kyma.dev" ]
        methods: ["GET"]
        paths: ["/headers"]
  selector:
    matchLabels:
      app: istio-ingressgateway
```

A request whose Host header includes the port `443` does not match the policy, so Istio denies it:

```bash
curl -k -H "Host: httpbin.local.kyma.dev:443" https://httpbin.local.kyma.dev/headers
```
```
RBAC: access denied
```

### Solution

To fix a `403` error caused by a port number in the Host header, follow these steps:

1. [RFC 9110](https://datatracker.ietf.org/doc/html/rfc9110#section-4.2.3) and [RFC 3986](https://datatracker.ietf.org/doc/html/rfc3986#section-3.2.3) define that HTTP clients remove the port if it is the default port for a given protocol. As a best practice, fix the client implementation.

2. If you cannot fix the client, modify the AuthorizationPolicy to also include the port number:

    ```yaml
    apiVersion: security.istio.io/v1
    kind: AuthorizationPolicy
    metadata:
      name: ingress-allow-headers
      namespace: istio-system
    spec:
      action: ALLOW
      rules:
      - to:
        - operation:
            hosts: [ "httpbin.local.kyma.dev", "httpbin.local.kyma.dev:443" ]
            methods: ["GET"]
            paths: ["/headers"]
      selector:
        matchLabels:
          app: istio-ingressgateway
    ```

If the port number is not the cause, inspect the AuthorizationPolicy resources that Istio Ingress Gateway enforces.

1. Check the mesh for over-broad policies. The analyzer reports an `IST0133` warning when a `DENY` policy uses only HTTP attributes and therefore blocks all TCP traffic in its scope:

    ```bash
    istioctl analyze --all-namespaces
    ```
    ```
    Warning [IST0133] (AuthorizationPolicy app-403/deny-no-port) Schema validation warning: configured AuthorizationPolicy will deny all traffic to TCP ports under its scope due to the use of only HTTP attributes in a DENY rule; it is recommended to explicitly specify the port
    ```

2. Print the AuthorizationPolicy list that the Envoy proxy applies to the Istio Ingress Gateway Pod. Unlike `kubectl get authorizationpolicies`, which lists the resources you created, this command shows the policies that Envoy actually enforces, merged from the mesh, namespace, and workload levels.

    Get the name of the Istio Ingress Gateway Pod:

    ```bash
    kubectl get pods --namespace istio-system --selector istio=ingressgateway --output name
    ```

    Check the policies applied to that Pod:

    ```bash
    istioctl x authz check {POD_NAME}.istio-system
    ```
    ```
    ACTION   AuthorizationPolicy                  RULES
    ALLOW    ingress-allow-headers.istio-system   1
    ```

    If an `ALLOW` policy applies, Envoy denies every request that does not match one of its rules. If a `DENY` policy applies, Envoy denies every request that matches it. Verify that the policy's `to`, `from`, and `when` rules cover the request that you send.

For more information about debugging AuthorizationPolicy resources, see [Network Connectivity - Diagnostics](./03-01-network-connectivity.md#authorizationpolicies).

## The 403 Comes from Your Application

If the request reaches your workload and the application returns the `403` error, Istio is not the cause. The proxy still adds the `server: istio-envoy` header, but the response also carries the `x-envoy-upstream-service-time` header and your application's own body.

### Cause

Istio routes the request correctly, but the application rejects it with its own `403` response, for example because of its own authentication or authorization logic. Because the response originates in the workload, no Istio configuration change fixes it.

The following response comes through Istio Ingress Gateway. The `server: istio-envoy` header is present, but so is the `x-envoy-upstream-service-time` header, and the body is the application's own payload:

```bash
curl -sD - http://REDACTED/forbidden
```
```
HTTP/1.1 403 Forbidden
server: istio-envoy
content-type: application/octet-stream
content-length: 29
x-envoy-upstream-service-time: 0

forbidden by the application
```

Compare this with a response from an AuthorizationPolicy, which has no `x-envoy-upstream-service-time` header and an `RBAC: access denied` body:

```
HTTP/1.1 403 Forbidden
content-length: 19
content-type: text/plain
server: istio-envoy

RBAC: access denied
```

### Solution

To confirm that the application returns the `403` error, bypass the mesh and send the request directly to the Pod. If the Pod still returns the `403` error, the application is the source, not Istio.

1. Forward a local port to the workload Pod:

    ```bash
    kubectl port-forward deployment/{DEPLOYMENT_NAME} 18080:{CONTAINER_PORT} --namespace {NAMESPACE}
    ```

2. In another terminal, send the request to the forwarded port and include the response headers:

    ```bash
    curl -sD - http://127.0.0.1:18080/{YOUR_PATH}
    ```
    ```
    HTTP/1.1 403 Forbidden
    Server: nginx/1.30.6

    forbidden by the application
    ```

    The `Server` header now shows the application server, such as `nginx`, instead of `istio-envoy`, because the request did not pass through the proxy. The `403` response still appears, which confirms that the application generates it.

Because the application returns the `403` error, debug the workload, not Istio. Check the application's authentication and authorization configuration, its logs, and any identity or token that it expects in the request.

## The 403 Comes from Outside the Mesh

If the `403` response has no `server: istio-envoy` header, or carries a vendor-specific error page instead of `RBAC: access denied`, something outside the mesh produced it.

### Cause

An external firewall, proxy, or WAF in front of the cluster rejects the request before it reaches Envoy. This is outside the scope of Istio.

### Solution

Because the block is outside the mesh, you cannot fix it with Istio configuration. Contact your network administrator, and provide the request details and the full response. To confirm that the traffic is blocked at the network layer rather than inside the mesh, see [Check for an External Firewall Blocking Access](./03-01-network-connectivity.md#check-for-an-external-firewall-blocking-access).
