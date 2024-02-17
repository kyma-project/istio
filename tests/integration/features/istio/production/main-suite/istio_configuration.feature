Feature: Configuration of Istio module

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Updating proxy resource configuration
    Given Template value "ProxyCPURequest" is set to "30m"
    And Template value "ProxyMemoryRequest" is set to "190Mi"
    And Template value "ProxyCPULimit" is set to "700m"
    And Template value "ProxyMemoryLimit" is set to "700Mi"
    And Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "test-app" deployment is created in namespace "default"
    And Application "test-app" in namespace "default" has proxy with "requests" set to cpu - "30m" and memory - "190Mi"
    And Application "test-app" in namespace "default" has proxy with "limits" set to cpu - "700m" and memory - "700Mi"
    And "Deployment" "test-app" in namespace "default" is ready
    And Template value "ProxyCPURequest" is set to "80m"
    And Template value "ProxyMemoryRequest" is set to "230Mi"
    And Template value "ProxyCPULimit" is set to "900m"
    And Template value "ProxyMemoryLimit" is set to "900Mi"
    When Istio CR "istio-sample" from "istio_cr_template" is updated in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    Then "Deployment" "test-app" in namespace "default" is ready
    And Application "test-app" in namespace "default" has proxy with "requests" set to cpu - "80m" and memory - "230Mi"
    And Application "test-app" in namespace "default" has proxy with "limits" set to cpu - "900m" and memory - "900Mi"

  Scenario: Ingress Gateway adds correct X-Envoy-External-Address header after updating numTrustedProxies
    Given Template value "NumTrustedProxies" is set to "1"
    And Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And "Deployment" "httpbin" in namespace "default" is ready
    And Istio gateway "test-gateway" is configured in namespace "default"
    And Virtual service "test-vs" exposing service "httpbin.default.svc.cluster.local" by gateway "default/test-gateway" is configured in namespace "default"
    And Request with header "X-Forwarded-For" with value "10.2.1.1,10.0.0.1" sent to httpbin should return "X-Envoy-External-Address" with value "10.0.0.1"
    When Template value "NumTrustedProxies" is set to "2"
    And Istio CR "istio-sample" from "istio_cr_template" is updated in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    Then Request with header "X-Forwarded-For" with value "10.2.1.1,10.0.0.1" sent to httpbin should return "X-Envoy-External-Address" with value "10.2.1.1"

  Scenario: External authorizer
    And Istio CR "istio-sample" from "istio_cr_ext_authz_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio injection is "enabled" in namespace "default"
    And Ext-authz application "ext-authz" deployment is created in namespace "default"
    And "Deployment" "ext-authz" in namespace "default" is ready
    And Httpbin application "httpbin" deployment is created in namespace "default"
    And "Deployment" "httpbin" in namespace "default" is ready
