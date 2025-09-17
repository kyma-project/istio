# Extend External Authorizer Definition in the Istio CR to Include pathPrefix and Timeout Configurations

## Status
<!--- Specify the current state of the ADR, such as whether it is proposed, accepted, rejected, deprecated, superseded, etc. -->
Accepted

## Context
<!--- Describe the issue or problem that is motivating this decision or change. -->
The current implementation of the External Authorizer in the Istio custom resource (CR) lacks the ability to specify a custom path prefix and timeout for the authorization requests.
This limitation can lead to challenges in integrating with external authorization services that require specific endpoints or have different performance characteristics.

## Decision
<!--- Explain the proposed change or action and the reason behind it. -->
To enhance the flexibility and usability of the External Authorizer in the Istio CR, we propose to extend its configuration options to include:
1. **pathPrefix**: Allows users to specify a custom path prefix for the authorization requests. For example, if the external authorizer service expects requests at `/auth/users`, while requests to the end service are sent to `/users`, the pathPrefix can be set to `/auth` to ensure correct routing.
2. **timeout**: Introduces a timeout setting to define how long the system should wait for a response from the external authorizer before considering the request as failed.

The above fields are exposed as a one-to-one mapping of the Istio Mesh Config configuration. They are optional, and if not set, the values fall back to Istio's defaults.

These additions enable better integration with a wider range of external authorization services and improve the overall reliability of the authorization process.

See the API structure after the change:
```yaml
apiVersion: istios.kyma-project.io/v1alpha1
kind: IstioCR
spec:
  config:
    authorizers:
      name: string (required)
      service: string (required)
      port: int  (required)
      headers: Headers (optional)
      pathPrefix: string (optional) # New field to specify the path prefix for the authorization requests
      timeout: Duration (optional) # New field to specify the timeout duration for the authorization requests
  # Other fields remain unchanged. Omitted for clarity
```
