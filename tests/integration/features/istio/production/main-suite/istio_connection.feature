Feature: Connection in the mesh

  Background:
    Given "Istio CR" is not present on cluster
    And Istio CRD is installed
    # We check that the namespaces used in the tests are not present, because clean up of the namespaces sometimes takes a little longer between tests
    And Namespace "source" is "not present"
    And Namespace "target" is "not present"
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Access between applications in different namespaces
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "target" is created
    And Istio injection is "enabled" in namespace "target"
    And Httpbin application "httpbin" is running in namespace "target" with service port "80"
    And Application pod "httpbin" in namespace "target" has Istio proxy "present"
    And Namespace "source" is created
    And Istio injection is "enabled" in namespace "source"
    And Nginx application "nginx" is running in namespace "source" with forward to "http://httpbin.target.svc.cluster.local/headers" and service port 80
    And Application pod "nginx" in namespace "source" has Istio proxy "present"
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "nginx.source.svc.cluster.local" with port "80" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request to path "/" should return "Host" with value "httpbin.target.svc.cluster.local" in body

  Scenario: Access between applications from injection disabled namespace to injection enabled namespace is restricted
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "target" is created
    And Istio injection is "enabled" in namespace "target"
    And Httpbin application "httpbin" is running in namespace "target" with service port "80"
    And Application pod "httpbin" in namespace "target" has Istio proxy "present"
    And Namespace "source" is created
    And Istio injection is "disabled" in namespace "source"
    And Nginx application "nginx" is running in namespace "source" with forward to "http://httpbin.target.svc.cluster.local/headers" and service port 80
    And Application pod "nginx" in namespace "source" has Istio proxy "not present"
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "nginx.source.svc.cluster.local" with port "80" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request to path "/" should have response code "502"