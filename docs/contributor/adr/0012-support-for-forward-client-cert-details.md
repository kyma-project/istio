# Extend Istio Custom Resource to Support **forwardClientCertDetails**

## Status
<!--- Specify the current state of the ADR, such as whether it is proposed, accepted, rejected, deprecated, superseded, etc. -->
Accepted

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->
To ensure that the client certificate details can be forwarded to the backend
services when mTLS is used to secure the Istio ingress gateway, 
it is essential to extend the Istio CustomResourceDefinitions (CRDs) to include a new field named **forwardClientCertDetails**.
This field allows to configure the strategy for forwarding client certificate information in the **X-Forwarded-Client-Cert** header.
For more information about the available strategies, see the official [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#enum-extensions-filters-network-http-connection-manager-v3-httpconnectionmanager-forwardclientcertdetails).

## Decision
<!--- Explain the proposed change or action and the reason behind it. -->
The Istio CRD will be extended to include a new optional field named **forwardClientCertDetails**.
This field will be added in the **spec.config** section.

The field will be of type string and will accept the following values:
- `APPEND_FORWARD` - When the client connection is mTLS, append the client certificate information to the request’s XFCC header and forward it.
- `SANITIZE_SET` - When the client connection is mTLS, reset the XFCC header with the client certificate information and send it to the next hop.
- `SANITIZE` - Do not send the XFCC header to the next hop.
- `ALWAYS_FORWARD_ONLY` - Always forward the XFCC header in the request, regardless of whether the client connection is mTLS.
- `FORWARD_ONLY` - When the client connection is mTLS, forward the XFCC header in the request.

The default value for this field will be `SANITIZE`, ensuring that, by default,
the XFCC header is not forwarded unless explicitly configured otherwise.
This default behaviour is defined by Envoy.
