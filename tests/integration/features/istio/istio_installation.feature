Feature: Installing and uninstalling Istio module

  Scenario: Installation of Istio module
    Given Istio CRD is installed
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready
    When Istio CR "istio-sample" is applied in namespace "default"
    Then Istio CR "istio-sample" in namespace "default" has status "Ready"
    And Namespace "istio-system" is "present"
    And Istio CRDs "should" be present on cluster
    And "Deployment" "istiod" in namespace "istio-system" is ready
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready

  Scenario: Uninstallation of Istio module
    Given Istio CR "istio-sample" in namespace "default" has status "Ready"
    When "Istio CR" "istio-sample" in namespace "default" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"
