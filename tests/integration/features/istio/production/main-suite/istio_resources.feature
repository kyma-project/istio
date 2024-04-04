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
    And "PeerAuthentication" "default" in namespace "istio-system" is "present"

    # Uninstalling Istio
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then "Istio CR" is not present on cluster
    And Namespace "istio-system" is "not present"
