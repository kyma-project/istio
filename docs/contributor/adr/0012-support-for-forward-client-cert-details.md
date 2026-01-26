# Extend Istio Custom Resource to Support **forwardClientCertDetails**

## Status
<!--- Specify the current state of the ADR, such as whether it is proposed, accepted, rejected, deprecated, superseded, etc. -->
Accepted

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->
To ensure that the client certificate details can be forwarded to the backend
services when mTLS is used to secure the Istio ingress gateway, 
it is essential to extend the Istio CustomResourceDefinitions (CRDs) to include a new field named **forwardClientCertDetails**.
This field allows to configure the strategy for forwarding client certificate information in the **X-Forwarded-Client-Cert** header for gateway proxies. This setting controls how the client attributes are retrieved from the incoming traffic by the gateway proxy and propagated to the upstream services in the cluster. By default, upstream Istio uses the `SANITIZE_SET` strategy for the gateway proxy, which resets the XFCC header with the client certificate information and sends it to the next hop.
For more information about the available strategies, see the official [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#enum-extensions-filters-network-http-connection-manager-v3-httpconnectionmanager-forwardclientcertdetails) and [Istio documentation](https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/#Topology-forward_client_cert_details).

## Decision
<!--- Explain the proposed change or action and the reason behind it. -->
The Istio CRD will be extended to include a new optional field named **forwardClientCertDetails**.
This field will be added in the **spec.config** section.

The field will be of type string and will accept the following values:
- `APPEND_FORWARD` - When the client connection is mTLS, append the client certificate information to the request’s XFCC header and forward it. This is the default value of istio upstream for sidecar proxies.
- `SANITIZE_SET` - When the client connection is mTLS, reset the XFCC header with the client certificate information and send it to the next hop. This is the default value of Istio upstream for gateway proxies.
- `SANITIZE` - Do not send the XFCC header to the next hop.
- `ALWAYS_FORWARD_ONLY` - Always forward the XFCC header in the request, regardless of whether the client connection is mTLS.
- `FORWARD_ONLY` - When the client connection is mTLS, forward the XFCC header in the request.

The field will not have a default value specified in the CRD schema, allowing users to explicitly set it based on their requirements.
The default value defined by Istio upstream for this field is `SANITIZE_SET` for gateway proxies, ensuring that, by default,
the XFCC header is reset and sent to the next hop, unless explicitly configured otherwise in Istio module CR.

