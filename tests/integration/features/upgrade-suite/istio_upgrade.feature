# Important: scenarios in this feature rely on the execution order
Feature: Upgrade Istio
  Scenario: Upgrade from latest release to current version
    Given "Istio CR" is not present on cluster
    And Istio CRD is installed
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready
    And Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "test-app" deployment is created in namespace "default"
    And Application pod "test-app" in namespace "default" has Istio proxy "present"
    And Istio gateway "test-gateway" is configured in namespace "default"
    And Virtual service "upgrade-test-vs" exposing service "test-app.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    And Request to path "/headers" should have response code "200"
    When Istio controller has been upgraded
    Then "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready
    And Istio CR "istio-sample" in namespace "kyma-system" status update happened in the last 20 seconds
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Application "test-app" in namespace "default" has required version of proxy
    And "Deployment" "istiod" in namespace "istio-system" is ready
    And Container "discovery" of "Deployment" "istiod" in namespace "istio-system" has required version
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And Container "istio-proxy" of "Deployment" "istio-ingressgateway" in namespace "istio-system" has required version
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready
    And Container "install-cni" of "DaemonSet" "istio-cni-node" in namespace "istio-system" has required version
