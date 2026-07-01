# You Get 403 Forbidden if Host Header Contains Port 

## Symptom

When you try to access a Kyma endpoint protected by AuthorizationPolicy allowing given host name, but it reports a 403 Forbidden error.

## Cause

The error might be caused by the unnecessary port number in the Host header. The Istio checks the host as-is, so if the Host header contains a port number, and AuthorizationPolicy defines only a host name then the request is denied.

Example:

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

```
curl -k -H "Host: httpbin.local.kyma.dev:443" https://httpbin.local.kyma.dev/headers
```

```
RBAC: access denied
```

## Solution

[RFC 9110](https://datatracker.ietf.org/doc/html/rfc9110#section-4.2.3) and [RFC 3986](https://datatracker.ietf.org/doc/html/rfc3986#section-3.2.3) documents define that HTTP clients should remove port if it is a default port for a given protocol. So the general recommendation is to fix the client implementation.

If above is not possible then the workaround is to adapt the AuthorizationPolicy to contain also the port number.

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
