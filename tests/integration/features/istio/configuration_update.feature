Feature: Updating configuration of Istio module

  Scenario: Updating proxy resource configuration
    Given Istio CRD is installed
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready
    And Template value "ProxyCPURequest" is set to "30m"
    And Template value "ProxyMemoryRequest" is set to "190Mi"
    And Template value "ProxyCPULimit" is set to "700m"
    And Template value "ProxyMemoryLimit" is set to "700Mi"
    And Istio CR "istio-sample" is applied in namespace "default"
    And Istio CR "istio-sample" in namespace "default" has status "Ready"
    And Istio injection is enabled in namespace "default"
    And Application "test-app" is running in namespace "default"
    And Application "test-app" in namespace "default" has proxy with "requests" set to cpu - "30m" and memory - "190Mi"
    And Application "test-app" in namespace "default" has proxy with "limits" set to cpu - "700m" and memory - "700Mi"
    And Template value "ProxyCPURequest" is set to "80m"
    And Template value "ProxyMemoryRequest" is set to "230Mi"
    And Template value "ProxyCPULimit" is set to "900m"
    And Template value "ProxyMemoryLimit" is set to "900Mi"
    When Istio CR "istio-sample" is updated in namespace "default"
    Then Application "test-app" in namespace "default" has proxy with "requests" set to cpu - "80m" and memory - "230Mi"
    And Application "test-app" in namespace "default" has proxy with "limits" set to cpu - "900m" and memory - "900Mi"
