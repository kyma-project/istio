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
    And Httpbin application "test-app-init-container" deployment with proxy as a native sidecar is created in namespace "default"
    And Application pod "test-app" in namespace "default" has Istio proxy "present"
    And Application "test-app-init-container" in namespace "default" has init container with Istio proxy present
    And Istio gateway "test-gateway" is configured in namespace "default"
    And Virtual service "upgrade-test-vs" exposing service "test-app.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    And Virtual service "upgrade-test-vs-init-container" exposing service "test-app-init-container.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    And Request with header "Host" with value "test-app.default.svc.cluster.local" to path "/headers" should have response code "200"
    And Request with header "Host" with value "test-app-init-container.default.svc.cluster.local" to path "/headers" should have response code "200"
    And There are continuous requests to host "test-app.default.svc.cluster.local" and path "/headers"
    And There are continuous requests to host "test-app-init-container.default.svc.cluster.local" and path "/headers"
    When Istio controller has been upgraded
    Then "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready
    And Istio CR "istio-sample" in namespace "kyma-system" status update happened in the last 20 seconds
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Application "test-app" in namespace "default" has required version of proxy
    And Application "test-app-init-container" in namespace "default" has required version of proxy as an init container
    And "Deployment" "istiod" in namespace "istio-system" is ready
    And Container "discovery" of "Deployment" "istiod" in namespace "istio-system" has required version
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And Container "istio-proxy" of "Deployment" "istio-ingressgateway" in namespace "istio-system" has required version
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready
    And Container "install-cni" of "DaemonSet" "istio-cni-node" in namespace "istio-system" has required version
    And All continuous requests should succeed
