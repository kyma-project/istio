Feature: Istio sidecar injection works properly in target namespace

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

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
