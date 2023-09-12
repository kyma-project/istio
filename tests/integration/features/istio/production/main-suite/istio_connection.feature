Feature: Connection in the mesh

  Background:
    Given "Istio CR" is not present on cluster
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Access between applications in different namespaces
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "target" is created
    And Istio injection is "enabled" in namespace "target"
    And Httpbin application "httpbin" is running in namespace "target" with service port "80"
    And Namespace "source" is created
    And Istio injection is "enabled" in namespace "source"
    And Nginx application "nginx" is running in namespace "source" with forward to "http://httpbin.target.svc.cluster.local/headers"
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "nginx.source.svc.cluster.local" with port "80" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request to path "/" should return "Host" with value "httpbin.target.svc.cluster.local" in body
