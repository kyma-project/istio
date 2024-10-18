Feature: Installing Istio module with evaluation profile
  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Evaluation"
    And Istio CRD is installed
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Installation of istio module with evaluation profile
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileSucceeded" of type "Ready" and status "True"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    When "Deployment" "httpbin" in namespace "default" is ready
    Then Pod of deployment "httpbin" in namespace "default" has container "istio-proxy" with resource "requests" set to cpu - "10m" and memory - "32Mi"
    And Pod of deployment "httpbin" in namespace "default" has container "istio-proxy" with resource "limits" set to cpu - "250m" and memory - "254Mi"
    And "ingress-gateway" has "requests" set to cpu - "10m" and memory - "32Mi"
    And "ingress-gateway" has "limits" set to cpu - "1000m" and memory - "1024Mi"
    And "pilot" has "requests" set to cpu - "50m" and memory - "128Mi"
    And "pilot" has "limits" set to cpu - "1000m" and memory - "1024Mi"
