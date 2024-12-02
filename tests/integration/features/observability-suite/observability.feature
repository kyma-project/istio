Feature: Observability

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready
    And Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio injection is "enabled" in namespace "default"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"

  Scenario: Logs from stdout-json envoyFileAccessLog provider are in correct format
    Given Access logging is enabled for the mesh using "stdout-json" provider
    And Istio gateway "test-gateway" is configured in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And Virtual service "httpbin" exposing service "httpbin.default.svc.cluster.local" with port "8000" by gateway "default/test-gateway" is configured in namespace "default"
    And Request to path "/ip" should have response code "200"
    Then Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "start_time"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "method"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "path"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "protocol"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "response_code"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "response_flags"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "response_code_details"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "connection_termination_details"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "upstream_transport_failure_reason"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "bytes_received"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "bytes_sent"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "duration"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "upstream_service_time"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "x_forwarded_for"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "user_agent"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "request_id"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "authority"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "upstream_host"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "upstream_cluster"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "upstream_local_address"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "downstream_local_address"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "downstream_remote_address"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "requested_server_name"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "route_name"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "traceparent"
    And Log of container "istio-proxy" in deployment "istio-ingressgateway" in namespace "istio-system" contains "tracestate"

  Scenario: Istio calls OpenTelemetry API on default service configured in kyma-traces extension provider
    Given Tracing is enabled for the mesh using provider "kyma-traces"
    # For a simpler setup we use a tcp-echo as OpenTelemetry collector mock, because we only want to verify that the OpenTelemetry API is called by checking the echoed request logs.
    And Istio gateway "test-gateway" is configured in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And Virtual service "httpbin" exposing service "httpbin.default.svc.cluster.local" with port "8000" by gateway "default/test-gateway" is configured in namespace "default"
    And OTEL Collector mock "otel-collector-mock" deployment is created in namespace "kyma-system"
    And Service is created for the otel collector "otel-collector-mock" in namespace "kyma-system"
    When Request to path "/ip" should have response code "200"
    Then Log of container "otel-collector-mock" in deployment "otel-collector-mock" in namespace "kyma-system" contains "POST /opentelemetry.proto.collector.trace.v1.TraceService/Export"