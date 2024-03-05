package v1alpha2

type Authorizer struct {
	// A unique name identifying the extension authorization provider.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Specifies the service that implements the Envoy ext_authz HTTP authorization service.
	// The format is "[<Namespace>/]<Hostname>".
	// The specification of "<Namespace>"
	// is required only when it is insufficient to unambiguously resolve a service in the service registry.
	// The "<Hostname>" is a fully qualified host name of a service defined by the Kubernetes service or ServiceEntry.
	// The recommended format is "[<Namespace>/]<Hostname>"
	// Example: "my-ext-authz.foo.svc.cluster.local" or "bar/my-ext-authz".
	// +kubebuilder:validation:Required
	Service string `json:"service"`

	// Specifies the port of the service.
	// +kubebuilder:validation:Required
	Port uint32 `json:"port"`

	// Specifies headers to be included, added or forwarded during authorization.
	Headers *Headers `json:"headers,omitempty"`
}

// Exact, prefix and suffix matches are supported (similar to the authorization policy rule syntax except the presence match
// https://istio.io/latest/docs/reference/config/security/authorization-policy/#Rule):
// - Exact match: "abc" will match on value "abc".
// - Prefix match: "abc*" will match on value "abc" and "abcd".
// - Suffix match: "*abc" will match on value "abc" and "xabc".

type Headers struct {
	// Defines headers to be included or added in check authorization request.
	InCheck *InCheck `json:"inCheck,omitempty"`

	// Defines headers to be forwarded to the upstream.
	ToUpstream *ToUpstream `json:"toUpstream,omitempty"`

	// Defines headers to be forwarded to the downstream.
	ToDownstream *ToDownstream `json:"toDownstream,omitempty"`
}

type InCheck struct {
	// List of client request headers that should be included in the authorization request sent to the authorization service.
	// Note that in addition to the headers specified here, the following headers are included by default:
	// 1. *Host*, *Method*, *Path* and *Content-Length* are automatically sent.
	// 2. *Content-Length* will be set to 0, and the request will not have a message body. However, the authorization request can include the buffered client request body (controlled by include_request_body_in_check setting), consequently the value of Content-Length of the authorization request reflects the size of its payload size.
	Include []string `json:"include,omitempty"`

	// Set of additional fixed headers that should be included in the authorization request sent to the authorization service.
	// The Key is the header name and value is the header value.
	// Note that client request of the same key or headers specified in `Include` will be overridden.
	Add map[string]string `json:"add,omitempty"`
}

type ToUpstream struct {
	// List of headers from the authorization service that should be added or overridden in the original request and forwarded to the upstream when the authorization check result is allowed (HTTP code 200).
	// If not specified, the original request will not be modified and forwarded to backend as-is.
	// Note, any existing headers will be overridden.
	OnAllow []string `json:"onAllow,omitempty"`
}

type ToDownstream struct {
	// List of headers from the authorization service that should be forwarded to downstream when the authorization check result is allowed (HTTP code 200).
	// If not specified, the original response will not be modified and forwarded to downstream as-is.
	// Note, any existing headers will be overridden.
	OnAllow []string `json:"onAllow,omitempty"`

	// List of headers from the authorization service that should be forwarded to downstream when the authorization check result is not allowed (HTTP code other than 200).
	// If not specified, all the authorization response headers, except *Authority (Host)* will be in the response to the downstream.
	// When a header is included in this list, *Path*, *Status*, *Content-Length*, *WWWAuthenticate* and *Location* are automatically added.
	// Note, the body from the authorization service is always included in the response to downstream.
	OnDeny []string `json:"onDeny,omitempty"`
}
