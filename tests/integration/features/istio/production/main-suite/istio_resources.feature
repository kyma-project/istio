Feature: Istio resources configuration

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Additional Istio resources are present
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    When Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And "EnvoyFilter" "kyma-referer" in namespace "istio-system" is "present"
    And "PeerAuthentication" "default" in namespace "istio-system" is "present"

    # Uninstalling Istio
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then "Istio CR" is not present on cluster
    And Namespace "istio-system" is "not present"

  Scenario: Ingress Gateway and proxy sidecar allow Referer Header with fragment identifier (# character)
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And Istio gateway "test-gateway" is configured in namespace "default"
    And "Deployment" "httpbin" in namespace "default" is ready
    When Virtual service "test-vs" exposing service "httpbin.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request with header "Referer" with value "https://someurl.com/context/#view" sent to httpbin should return "Referer" with value "https://someurl.com/context/#view"
