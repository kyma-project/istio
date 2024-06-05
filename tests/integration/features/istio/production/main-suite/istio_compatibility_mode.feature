Feature: Compatibility Mode

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Compatibility mode is set on
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    # ENABLE_EXTERNAL_NAME_ALIAS is set to true by the custom annotation handler if the compatibility mode is off, so we check default setup here
    Then Environment variable "ENABLE_EXTERNAL_NAME_ALIAS" on Deployment "istiod" in namespace "istio-system" is present and has value "true"
    And Environment variable "VERIFY_CERTIFICATE_AT_CLIENT" on Deployment "istiod" in namespace "istio-system" is not present
    And Environment variable "ENABLE_AUTO_SNI" on Deployment "istiod" in namespace "istio-system" is not present
    And Environment variable "PERSIST_OLDEST_FIRST_HEURISTIC_FOR_VIRTUAL_SERVICE_HOST_MATCHING" on Deployment "istiod" in namespace "istio-system" is not present
    Then Template value "CompatibilityMode" is set to "true"
    And Istio CR "istio-sample" from "istio_cr_template" is updated in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    Then Environment variable "ENABLE_EXTERNAL_NAME_ALIAS" on Deployment "istiod" in namespace "istio-system" is present and has value "false"
    And Environment variable "VERIFY_CERTIFICATE_AT_CLIENT" on Deployment "istiod" in namespace "istio-system" is present and has value "false"
    And Environment variable "ENABLE_AUTO_SNI" on Deployment "istiod" in namespace "istio-system" is present and has value "false"
    And Environment variable "PERSIST_OLDEST_FIRST_HEURISTIC_FOR_VIRTUAL_SERVICE_HOST_MATCHING" on Deployment "istiod" in namespace "istio-system" is present and has value "true"
