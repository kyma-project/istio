Feature: Observability

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready
    And Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio gateway "test-gateway" is configured in namespace "default"
    And Istio injection is "enabled" in namespace "default"

  Scenario: Logs from ingress-gateway and the sidecar are present and in correct format
    Given Access logging is enabled for the mesh using "stdout-json" provider
    And Httpbin application "httpbin" is running in namespace "default"
    And Virtual service "httpbin" exposing service "httpbin.default.svc.cluster.local" with port "8000" by gateway "default/test-gateway" is configured in namespace "default"
    And Request to path "/ip" should have response code "200"
    Then Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "start_time"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "method"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "path"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "protocol"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "response_code"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "response_flags"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "response_code_details"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "connection_termination_details"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "upstream_transport_failure_reason"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "bytes_received"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "bytes_sent"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "duration"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "upstream_service_time"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "x_forwarded_for"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "user_agent"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "request_id"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "authority"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "upstream_host"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "upstream_cluster"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "upstream_local_address"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "downstream_local_address"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "downstream_remote_address"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "requested_server_name"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "route_name"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "traceparent"
    And Envoy logs of deployment "istio-ingressgateway" in namespace "istio-system" contains access log entry with "tracestate"

  Scenario: Istio interacts with the otel collector service defined in meshconfig
    Given Logging and tracing is enabled for the mesh using providers "stdout-json" for logs and "kyma-traces" for traces
    And Httpbin application "otel-collector-mock" is running in namespace "kyma-system"
    And Service is created for the otel collector "otel-collector" in namespace "kyma-system"
    Then Envoy logs of deployment "otel-collector-mock" in namespace "kyma-system" contains access log entry with "method"
    Then Envoy logs of deployment "otel-collector-mock" in namespace "kyma-system" contains access log entry with "tracestate"