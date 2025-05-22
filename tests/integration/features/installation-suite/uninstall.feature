Feature: Uninstall Istio module

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Uninstallation of Istio module
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileSucceeded" of type "Ready" and status "True"
    And Namespace "istio-system" is "present"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "test-app" deployment is created in namespace "default"
    And Application pod "test-app" in namespace "default" has Istio proxy "present"
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"
    And Application pod "test-app" in namespace "default" has Istio proxy "not present"

  Scenario: Uninstallation respects the Istio resources created by the user
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileSucceeded" of type "Ready" and status "True"
    And Namespace "istio-system" is "present"
    And Destination rule "customer-destination-rule" in namespace "default" with host "testing-svc.default.svc.cluster.local" exists
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then Istio CR "istio-sample" in namespace "kyma-system" has status "Warning"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "IstioCustomResourcesDangling" of type "Ready" and status "False"
    And Istio CR "istio-sample" in namespace "kyma-system" contains description "There are Istio resources that block deletion. Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning"
    And Istio CRDs "should" be present on cluster
    And Namespace "istio-system" is "present"
    When "DestinationRule" "customer-destination-rule" in namespace "default" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"

  Scenario: Uninstallation of Istio module if Istio was manually deleted
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileSucceeded" of type "Ready" and status "True"
    And Namespace "istio-system" is "present"
    And Istio is manually uninstalled
    And Namespace "istio-system" is "not present"
    And Istio CRDs "should not" be present on cluster
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"
