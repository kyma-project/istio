# Extend Istio CustomResource to support forwardClientCertDetails

## Status
<!--- Specify the current state of the ADR, such as whether it is proposed, accepted, rejected, deprecated, superseded, etc. -->
Accepted

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->
To ensure that when mTLS is used to secure the Istio ingress gateway,
the client certificate details can be forwarded to the backend services,
it is essential to extend the Istio CustomResource definitions to include a new field named `forwardClientCertDetails`.
This field allows to configure the strategy for forwarding client certificate information in the `X-Forwarded-Client-Cert` header.
More information about the available strategies can be found in the official [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#enum-extensions-filters-network-http-connection-manager-v3-httpconnectionmanager-forwardclientcertdetails).

## Decision
<!--- Explain the proposed change or action and the reason behind it. -->
The Istio CustomResourceDefinition will be extended to include a new optional field named `forwardClientCertDetails`.
This field will be added under the `spec.config` section.

The field will be of type string and will accept the following values:
- `AppendForward` - When the client connection is mTLS, append the client certificate information to the request’s XFCC header and forward it.
- `SanitizeSet` - When the client connection is mTLS, reset the XFCC header with the client certificate information and send it to the next hop.
- `Sanitize` - Do not send the XFCC header to the next hop.
- `AlwaysForwardOnly` - Always forward the XFCC header in the request, regardless of whether the client connection is mTLS.
- `ForwardOnly` - When the client connection is mTLS (Mutual TLS), forward the XFCC header in the request.

The default value for this field will be `Sanitize`, ensuring that by default,
the XFCC header is not forwarded unless explicitly configured otherwise.
This default behaviour is defined by Envoy.
