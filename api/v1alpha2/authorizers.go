package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Defines an external authorization provider's configuration.
// The defined authorizer can be referenced by name in an AuthorizationPolicy
// with action CUSTOM to enforce requests to be authorized by the external authorization service.
type Authorizer struct {
	// Specifies a unique name identifying the authorization provider.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Specifies the service that implements the Envoy `ext_authz` HTTP authorization service.
	// The recommended format is `[Namespace/]Hostname`.
	// Specify the namespace if it is required to unambiguously resolve a service in the service registry. 
	// The host name refers to the fully qualified host name of a service defined by either a Kubernetes Service or a ServiceEntry.
	Service string `json:"service"`

	// Specifies the port of the Service.
	// +kubebuilder:validation:Required
	Port uint32 `json:"port"`

	// Specifies the headers included, added, or forwarded during authorization.
	Headers *Headers `json:"headers,omitempty"`

	// Specifies the prefix included in the request sent to the authorization service.
	// The prefix might be constructed with special characters (for example, `/test?original_path=`).
	// +kubebuilder:validation:Optional
	PathPrefix *string `json:"pathPrefix,omitempty"`

	// Specifies the timeout for the HTTP authorization request to the external service.
	// +kubebuilder:validation:Optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// Specifies headers included, added, or forwarded during authorization.
// Exact, prefix, and suffix matches are supported, similar to the syntax used in AuthorizationPolicy rules (excluding the presence match):
// - Exact match: `abc` matches the value `abc`.
// - Prefix match: `abc*` matches the values `abc` and `abcd`.
// - Suffix match: `*abc` matches the values `abc` and `xabc`.
type Headers struct {
	// Defines the headers to be included or added in check authorization request.
	InCheck *InCheck `json:"inCheck,omitempty"`

	// Defines the headers to be forwarded to the upstream (to the backend service).
	ToUpstream *ToUpstream `json:"toUpstream,omitempty"`

	// Defines the headers to be forwarded to the downstream (the client).
	ToDownstream *ToDownstream `json:"toDownstream,omitempty"`
}

// Defines the headers to be included or added in check authorization request.
type InCheck struct {
	// Lists client request headers included in the authorization request sent to the authorization service.
	// In addition to the headers specified here, the following headers are included by default:
	// - *Host*, *Method*, *Path*, and *Content-Length* are automatically sent.
	// - *Content-Length* is set to `0`, and the request doesn't have a message body. However, the authorization request can include the buffered client request body (controlled by the **include_request_body_in_check** setting), consequently the **Content-Length** value of the authorization request reflects its payload size.
	Include []string `json:"include,omitempty"`

	// Specifies a set of additional fixed headers included in the authorization request sent to the authorization service.
	// The key is the header name and value is the header value.
	// Client request of the same key or headers specified in `Include` are overridden.
	Add map[string]string `json:"add,omitempty"`
}

// Defines the headers to be forwarded to the upstream (to the backend service).	
type ToUpstream struct {
	// Lists headers from the authorization service added or overridden in the original request and forwarded to the upstream when the authorization check result is allowed (HTTP code `200`).
	// If not specified, the original request is forwarded to the backend unmodified.
	// Any existing headers are overridden.
	OnAllow []string `json:"onAllow,omitempty"`
}

// Defines the headers to be forwarded to the downstream (the client).
type ToDownstream struct {
	// Lists headers from the authorization service forwarded to downstream when the authorization check result is allowed (HTTP code `200`).
	// If not specified, the original request is forwarded to the backend unmodified.
	// Any existing headers are overridden.
	OnAllow []string `json:"onAllow,omitempty"`

	// Lists headers from the authorization service forwarded to downstream when the authorization check result is not allowed (HTTP code is other than `200`).
	// If not specified, all the authorization response headers, except *Authority (Host)*, are included in the response to the downstream.
	// When a header is included in this list, the following headers are automatically added: *Path*, *Status*, *Content-Length*, *WWWAuthenticate*, and *Location*.
	// The body from the authorization service is always included in the response to downstream.
	OnDeny []string `json:"onDeny,omitempty"`
}