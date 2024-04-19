# Configurable Log Formats in Default Access Logs Providers

## Status

Accepted

## Context
<!--- What is the issue that we're seeing that is motivating this decision or change? -->
Users have custom request and response headers that they use in various solutions. They would like these headers to be also included in the access logs.
This ADR proposes making the access logging a configurable feature, which can support at least additional request and response headers. In the best case, users should be able to define their own log format.

### User Scenario

1. The user creates an Istio custom resource (CR) with the customized **config** field containing the **accessLog** field.
2. The module reads the configuration and alters the default `kyma-default-logger` and `kyma-default-otel-logger` based on the user-provided configuration.

## Decision

* We define the default configuration of static ExtensionProvider `EnvoyFileAccessLogProvider` with the name `kyma-default-logger`. It replaces the stdout-json extension.
* We define the default configuration of static ExtensionProvider `EnvoyOpenTelemetryLogProvider` with the name `kyma-default-otel-logger`.
* We extend the Istio CR **config** field to include configuration for the default `kyma-default-logger` and `kyma-default-otel-logger`.
* The user can define whether to replace the default configuration with the labels provided or merge the `labels` maps using the optional **strategy** field. By default, `merge` is the default strategy if the field is not specified in the CR.
    * `merge` adds additional labels defined by the user to the default labels and replaces values in the existing default keys if the user has defined the key with a custom value.
    * `replace` replaces all default label fields with user-provided fields.
* The `accessLog` field applies log format to both `kyma-default-logger` and `kyma-default-otel-logger`.
* If the `accessLog` field is not defined, the module applies default settings defined in the code.
* If the `accessLog` field is defined, but the `labels` field doesn't contain any keys or is not defined, apply the default configuration defined in the code and generate the `Warning` status in the CR.
* If any of the keys in the **labels** field has an empty value, ignore the key during the injection and generate the `Warning` status in the CR.

### Examples

Extended the Istio CR that contains the **accessLog** field.

#### Using the `merge` Strategy
```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    accessLog:
      strategy: merge
      labels:
        tenant: "%REQ(:TENANT)%"
```
The access logs then contain list of default values with merged custom `tenant` value, as follows:
```
{"tenant": "tenant-xxx", [default values]}
```

#### Using the `replace` Strategy

```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    accessLog:
      strategy: replace
      labels:
        tenant: "%REQ(:TENANT)%"
```
The access logs then contain a single field as follows:
```
{"tenant": "tenant-xxx"}
```
### Code Implementation Example

The `IstioConfig` struct is extended with `AccessLogConfig`.
```go
type AccessLogConfig struct {
  // Strategy defines how the Labels should be handled. Defaults to "merge". Optional.
  Strategy string `yaml:"strategy,omitempty"`
  // Labels contains a map of structured keys and associated values. Envoy command operators can be used.
  Labels map[string]string `yaml:"labels,omitempty"`
}
```

The Istio module defines the **DefaultAccessLogConfig** variable of type `AccessLogConfig` that contains the default access log configuration:
```go
var defaultAccessLogConfig = AccessLogConfig {
    Strategy: "merge",
    Labels: map[string]string{
           "start_time": "%START_TIME%",
            "method": "%REQ(:METHOD)%",
            "path": "%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%",
            "protocol": "%PROTOCOL%",
            "response_code": "%RESPONSE_CODE%",
            "response_flags": "%RESPONSE_FLAGS%",
            "response_code_details": "%RESPONSE_CODE_DETAILS%",
            "connection_termination_details": "%CONNECTION_TERMINATION_DETAILS%",
            "upstream_transport_failure_reason": "%CONNECTION_TERMINATION_DETAILS%",
            "bytes_received": "%BYTES_RECEIVED%",
            "bytes_sent": "%BYTES_SENT%",
            "duration": "%DURATION%",
            "upstream_service_time": "%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%",
            "x_forwarded_for": "%REQ(X-FORWARDED-FOR)%",
            "user_agent": "%REQ(USER-AGENT)%",
            "request_id": "%REQ(X-REQUEST-ID)%",
            "authority": "%REQ(:AUTHORITY)%",
            "upstream_host": "%UPSTREAM_HOST%",
            "upstream_cluster": "%UPSTREAM_CLUSTER%",
            "upstream_local_address": "%UPSTREAM_LOCAL_ADDRESS%",
            "downstream_local_address": "%DOWNSTREAM_LOCAL_ADDRESS%",
            "downstream_remote_address": "%DOWNSTREAM_REMOTE_ADDRESS%",
            "requested_server_name": "%REQUESTED_SERVER_NAME%",
            "route_name": "%ROUTE_NAME%",
            "traceparent": "%REQ(TRACEPARENT)%",
            "tracestate": "%REQ(TRACESTATE)%",
    }
```

## Consequences

* This feature will allow configuration of the default access log extension provider per instance, allowing user options more tailored to their needs without introducing breaking changes in the existing IstioOperator configuration.
* When using the `merge` strategy, the user will be allowed to modify default values by defining a field with the same key as the one used in the default configuration.
* The logging format will be standardized for both the OTEL and FileAccess providers and the same strategy will be applied to both providers.