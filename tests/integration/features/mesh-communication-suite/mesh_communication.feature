Feature: Connection in the Istio mesh

  Background:
    Given "Istio CR" is not present on cluster
    And Istio CRD is installed
    # We check that the namespaces used in the tests are not present, because clean up of the namespaces sometimes takes a little longer between tests
    And Namespace "source" is "not present"
    And Namespace "target" is "not present"
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Access between applications in different namespaces
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "target" is created
    And Istio injection is "enabled" in namespace "target"
    And Httpbin application "httpbin" deployment is created in namespace "target" with service port "80"
    And Application pod "httpbin" in namespace "target" has Istio proxy "present"
    And Namespace "source" is created
    And Istio injection is "enabled" in namespace "source"
    And Nginx application "nginx" deployment is created in namespace "source" with forward to "http://httpbin.target.svc.cluster.local/headers" and service port 80
    And Application pod "nginx" in namespace "source" has Istio proxy "present"
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "nginx.source.svc.cluster.local" with port "80" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request to host "nginx.source.svc.cluster.local" and path "/" should return "Host" with value "httpbin.target.svc.cluster.local" in body

  Scenario: Access between applications from injection disabled namespace to injection enabled namespace is restricted
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "target" is created
    And Istio injection is "enabled" in namespace "target"
    And Httpbin application "httpbin" deployment is created in namespace "target" with service port "80"
    And Application pod "httpbin" in namespace "target" has Istio proxy "present"
    And Namespace "source" is created
    And Istio injection is "disabled" in namespace "source"
    And Nginx application "nginx" deployment is created in namespace "source" with forward to "http://httpbin.target.svc.cluster.local/headers" and service port 80
    And Application pod "nginx" in namespace "source" has Istio proxy "not present"
    And Istio gateway "test-gateway" is configured in namespace "default"
    When Virtual service "test-vs" exposing service "nginx.source.svc.cluster.local" with port "80" by gateway "default/test-gateway" is configured in namespace "default"
    Then Request with header "Host" with value "nginx.source.svc.cluster.local" to path "/" should have response code "502"

  Scenario: Namespace with istio-injection=disabled label does not contain pods with istio sidecar
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    When Namespace "sidecar-disable" is created
    And Namespace "sidecar-disable" is "present"
    And Istio injection is "disabled" in namespace "sidecar-disable"
    And Httpbin application "test-app" deployment is created in namespace "sidecar-disable"
    Then Application pod "test-app" in namespace "sidecar-disable" has Istio proxy "not present"
    And "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    And "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Application pod "test-app" in namespace "sidecar-disable" has Istio proxy "not present"
    And "Namespace" "sidecar-disable" is deleted

  Scenario: Namespace with istio-injection=enabled label does contain pods with istio sidecar
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    When Namespace "sidecar-enable" is created
    And Namespace "sidecar-enable" is "present"
    And Istio injection is "enabled" in namespace "sidecar-enable"
    And Httpbin application "test-app" deployment is created in namespace "sidecar-enable"
    Then Application pod "test-app" in namespace "sidecar-enable" has Istio proxy "present"
    And "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    And "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Application pod "test-app" in namespace "sidecar-enable" has Istio proxy "not present"
    And "Namespace" "sidecar-enable" is deleted

  Scenario: Kube-system namespace does not contain pods with sidecar
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Httpbin application "test-app" deployment is created in namespace "kube-system"
    Then Application pod "test-app" in namespace "kube-system" has Istio proxy "not present"
    And "Deployment" "test-app" in namespace "kube-system" is deleted
