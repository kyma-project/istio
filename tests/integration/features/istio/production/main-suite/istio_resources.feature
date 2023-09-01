Feature: Istio resources configuration

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Additional Istio resources are present
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    When Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    Then "Gateway" "kyma-gateway" in namespace "kyma-system" is present
    And "EnvoyFilter" "kyma-referer" in namespace "istio-system" is present
    And "PeerAuthentication" "default" in namespace "istio-system" is present
    And "VirtualService" "istio-healthz" in namespace "istio-system" is present
    And "ConfigMap" "istio-control-plane-grafana-dashboard" in namespace "kyma-system" is present
    And "ConfigMap" "istio-mesh-grafana-dashboard" in namespace "kyma-system" is present
    And "ConfigMap" "istio-performance-grafana-dashboard" in namespace "kyma-system" is present
    And "ConfigMap" "istio-service-grafana-dashboard" in namespace "kyma-system" is present
    And "ConfigMap" "istio-workload-grafana-dashboard" in namespace "kyma-system" is present

  Scenario: Ingress Gateway and proxy sidecar allow Referer Header with fragment identifier (# character)
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" is running in namespace "default"
    And Istio gateway "test-gateway" is configured in namespace "default"
    And "Deployment" "httpbin" in namespace "default" is ready
    When Virtual service "test-vs" exposing service "httpbin.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request with header "Referer" with value "https://someurl.com/context/#view" sent to httpbin should return "Referer" with value "https://someurl.com/context/#view"
