Feature: X-Forwarded-For header

  Background:
    Given "Istio CR" is not present on cluster
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: X-Forward-For header contains public client IP when externalTrafficPolicy is not set in the IstioCR
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And "Deployment" "httpbin" in namespace "default" is ready
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "httpbin.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request sent to exposed httpbin, "should" contain public client IP in "X-Forwarded-For" header

  Scenario: X-Forward-For header contains public client IP when externalTrafficPolicy is set to Local
    Given Istio CR "istio-sample" is applied in namespace "kyma-system" with externalTrafficPolicy set to "Local"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And "Deployment" "httpbin" in namespace "default" is ready
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "httpbin.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request sent to exposed httpbin, "should" contain public client IP in "X-Forwarded-For" header

  Scenario: X-Forward-For header contains public client IP when externalTrafficPolicy is set to Cluster
    Given Istio CR "istio-sample" is applied in namespace "kyma-system" with externalTrafficPolicy set to "Local"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And "Deployment" "httpbin" in namespace "default" is ready
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "httpbin.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request sent to exposed httpbin, "should" contain public client IP in "X-Forwarded-For" header
