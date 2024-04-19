# Support for External Authorization Providers

## Status
Accepted

## Context
We are extending Istio CR by adding support for external authorization providers. This proposal is for authorization services that operate only over HTTP.

## Decision

* We define the API specification in such a way that it supports the current Istio mesh configuration for [EnvoyExternalAuthorizationHttpProvider](https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/#MeshConfig-ExtensionProvider-EnvoyExternalAuthorizationHttpProvider), which is one of the Istio's [extension providers](https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/#MeshConfig-ExtensionProvider). We exclude some properties that we think will not be used in Kyma scenarios to simplify the customer experience and reduce complexity in the UI. As a result, when defining an external authorizer, users only need to configure three required properties - **name**, **service**, and **port** - while having the optional **headers** property defined.

* We add support only for HTTP external authorization providers. This is because we expect the usage of external authentication providers to be a rare use case, and gRPC usage would be even more uncommon. Currently, in production, we rarely see customers using `oauth2_introspection`. That is why we do not consider other protocols (like gRPC) relevant at this moment.

* We support multiple external authorization providers as there might be use cases for this, such as different providers having unique capabilities, differentiating by authentication flows (needing different Deployments), multi-tenancy cases where an application needs to support multiple identity providers having a different provider per tenant or user pool. We cannot assume that an authorizer will work with multiple identity providers (for example, oauth2-proxy only supports this in alpha right now). Istio currently supports configuration for multiple [extension providers](https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/#MeshConfig-ExtensionProvider), which can be identified by a unique name.

* We do not include `oauth2-proxy` Deployment with Kyma-managed Istio installation. After analysis of the current project state, we did not see many recent contributions. Very long release cycles will make the handling of security vulnerabilities very difficult and, from past experiences managing ORY Hydra/Oathkeeper Deployments, we want to avoid handling similar external components in the future.

* For the optional headers, we see use cases where the user may need to adapt headers in the auth flow. Istio guides do include examples with headers (for example `oauth2-proxy`) and different identity providers may require specific headers.

* For the sake of simplicity, we do not include configuration properties for **Timeout**, **PathPrefix**, **FailOpen**, **StatusOnError**, and **IncludeRequestBodyIncheck** as in Istio's mesh configuration for [EnvoyExternalAuthorizationHttpProvider](https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/#MeshConfig-ExtensionProvider-EnvoyExternalAuthorizationHttpProvider). We don't see a use case for them and will rely on their default values.


### User Scenario / Flow (Example)

1. The customer installs Kyma with Istio and API Gateway.
2. Manual installation of `oauth2-proxy`. For example, see the [installation documentation](https://github.com/oauth2-proxy/manifests/tree/main/helm/oauth2-proxy).
3. Configure Istio CR with an external authorization provider for `oauth2-proxy` (see the example below).
4. The Istio module operator configures Istio with the additional mesh configuration and creates `ServiceEntry` (networking.istio.io) for it to be accessible within the service mesh (see example below).
5. The user may need to create manually `DestinationRule` (networking.istio.io) for an external OIDC provider (for example, SAP IAS).
6. Additional `VirtualService` (networking.istio.io) resources must be created by the customer for the `auth code flow` or `client credentials` use cases.
7. Expose the service via `APIRule` specifying a new authorization handler which the API Gateway module operator will reconcile and create an `AuthorizationPolicy` (security.istio.io) with `CUSTOM` action (see example below).

### Examples

* Istio CR with external provider configuration for `oauth2-proxy`:

```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    numTrustedProxies: 1
    authorizers:
    - name: "oauth2-proxy" # required, unique identifier
      service: "oauth2-proxy.oauth2-proxy.svc.cluster.local" # required
      port: "4180" # required
      headers: # optional
        inCheck:
          include: ["authorization", "cookie"] # headers sent to the oauth2-proxy in the check request.
          add:
            x-auth-request-redirect: "https://%REQ(:authority)%%REQ(x-envoy-original-path?:path)%"
        toUpstream:
          onAllow: ["authorization", "path", "x-auth-request-user", "x-auth-request-email", "x-auth-request-access-token"] # headers sent to backend application when request is allowed.
        toDownstream:
          onAllow: ["set-cookie"] # headers sent back to the client when request is allowed.
          onDeny: ["content-type", "set-cookie"] # headers sent back to the client when request is denied.
```

* `ServiceEntry` resource created during Istio module operator reconciliation:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: oauth2-proxy
  namespace: oauth2-proxy
spec:
  hosts:
  - oauth2-proxy.oauth2-proxy.svc.cluster.local
  ports:
  - name: http
    number: 4180
    protocol: http
  resolution: STATIC
```

* `APIRule` with new authorization handler:

```yaml
apiVersion: gateway.kyma-project.io/v1beta1
kind: APIRule
metadata:
  name: httpbin
spec:
  host: httpbin.xxx.shoot.canary.k8s-hana.ondemand.com
  gateway: kyma-system/kyma-gateway
  service:
    name: httpbin
    port: 8000
  rules:
    - path: /ip
      methods: ["GET"]
      accessStrategies:
        - handler: extern
          config:
            authorizerName: oauth2-proxy
```

* `AuthorizationPolicy` resource created during API Gateway module operator reconciliation for the `APIRule`:

```yaml
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: httpbin-abcde
spec:
  action: CUSTOM
  provider:
    name: oauth2-proxy
  rules:
  - to:
    - operation:
        methods: ["GET"]
        paths: ["/ip"]
  selector:
    matchLabels:
      app: httpbin
```

### API `golang` Structure Changes:

```go
type Config struct {
  // OPTIONAL. Defines the number of trusted proxies deployed in front of the Istio gateway proxy.
  // +kubebuilder:validation:Minimum=0
  // +kubebuilder:validation:Maximum=4294967295
  NumTrustedProxies *int `json:"numTrustedProxies,omitempty"`

  // OPTIONAL. Defines a list of external authorization providers.
  Authorizers []*Authorizer `json:"authorizers,omitempty"`
}

type Authorizer struct {
  // REQUIRED. A unique name identifying the extension authorizationprovider.
  Name string `json:"name,omitempty"`

  // REQUIRED. Specifies the service that implements the Envoy ext_authz HTTP authorization service.
  // The format is `[<Namespace>/]<Hostname>`. The specification of `<Namespace>` is required only when it is insufficient to unambiguously resolve a service in the service registry. The `<Hostname>` is a fully qualified host name of a service defined by the Kubernetes service or ServiceEntry.
  //
  // Example: "my-ext-authz.foo.svc.cluster.local" or "bar/my-ext-authz.example.com".
  Service string `json:"service,omitempty"`

  // REQUIRED. Specifies the port of the service.
  Port uint32 `json:"port,omitempty"`

  // OPTIONAL. Specifies headers to be included, added or forwarded during authorazation.
  Headers *Headers `json:"headers,omitempty"`
}

// Exact, prefix and suffix matches are supported (similar to the authorization policy rule syntax except the presence match
// https://istio.io/latest/docs/reference/config/security/authorization-policy/#Rule):
// - Exact match: "abc" will match on value "abc".
// - Prefix match: "abc*" will match on value "abc" and "abcd".
// - Suffix match: "*abc" will match on value "abc" and "xabc".

type Headers struct {
  // OPTIONAL. Defines headers to be included or added in check authorization request.
  InCheck *InCheck `json:"inCheck,omitempty"`

  // OPTIONAL. Defines headers to be forwarded to the upstream.
  ToUpstream *ToUpstream `json:"toUpstream,omitempty"`

  // OPTIONAL. Defines headers to be forwarded to the downstream.
  ToDownstream *ToDownstream `json:"toDownstream,omitempty"`
}

type InCheck struct {
  // OPTIONAL. List of client request headers that should be included in the authorization request sent to the authorization service.
  // Note that in addition to the headers specified here following headers are included by default:
  // 1. *Host*, *Method*, *Path* and *Content-Length* are automatically sent.
  // 2. *Content-Length* will be set to 0 and the request will not have a message body. However, the authorization request can include the buffered client request body (controlled by include_request_body_in_check setting), consequently the value of Content-Length of the authorization request reflects the size of its payload size.
  Include []string `json:"include,omitempty"`

  // OPTIONAL. Set of additional fixed headers that should be included in the authorization request sent to the authorization service.
  // Key is the header name and value is the header value.
  // Note that client request of the same key or headers specified in `Include` will be overridden.
  Add map[string]string `json:"add,omitempty"`
}

type ToUpstream struct {
  // OPTIONAL. List of headers from the authorization service that should be added or overridden in the original request and forwarded to the upstream when the authorization check result is allowed (HTTP code 200).
  // If not specified, the original request will not be modified and forwarded to backend as-is.
  // Note, any existing headers will be overridden.
  OnAllow []string `json:"onAllow,omitempty"`
}

type ToDownstream struct {
  // OPTIONAL. List of headers from the authorization service that should be forwarded to downstream when the authorization check result is allowed (HTTP code 200).
  // If not specified, the original response will not be modified and forwarded to downstream as-is.
  // Note, any existing headers will be overridden.
  OnAllow []string `json:"onAllow,omitempty"`

  // OPTIONAL. List of headers from the authorization service that should be forwarded to downstream when the authorization check result is not allowed (HTTP code other than 200).
  // If not specified, all the authorization response headers, except *Authority (Host)* will be in the response to the downstream.
  // When a header is included in this list, *Path*, *Status*, *Content-Length*, *WWWAuthenticate* and *Location* are automatically added.
  // Note, the body from the authorization service is always included in the response to downstream.
  OnDeny []string `json:"onDeny,omitempty"`
}
```

## Consequences

We add support for external authorization providers with Kyma Istio and do not introduce any breaking changes.
