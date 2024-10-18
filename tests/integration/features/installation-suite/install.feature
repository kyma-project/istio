Feature: Installing and uninstalling Istio module

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Installation of Istio module with default values
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileUnknown" of type "Ready" and status "Unknown"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileSucceeded" of type "Ready" and status "True"
    And Istio injection is "enabled" in namespace "default"
    And Httpbin application "httpbin" deployment is created in namespace "default"
    When "Deployment" "httpbin" in namespace "default" is ready
    Then Pod of deployment "httpbin" in namespace "default" has container "istio-proxy" with resource "requests" set to cpu - "10m" and memory - "192Mi"
    And Pod of deployment "httpbin" in namespace "default" has container "istio-proxy" with resource "limits" set to cpu - "1000m" and memory - "1024Mi"
    And "ingress-gateway" has "requests" set to cpu - "100m" and memory - "128Mi"
    And "ingress-gateway" has "limits" set to cpu - "2000m" and memory - "1024Mi"
    And "pilot" has "requests" set to cpu - "100m" and memory - "512Mi"
    And "pilot" has "limits" set to cpu - "4000m" and memory - "2048Mi"

  Scenario: Installation of Istio module
    Given Template value "PilotCPULimit" is set to "1200m"
    And Template value "PilotMemoryLimit" is set to "1200Mi"
    And Template value "PilotCPURequests" is set to "15m"
    And Template value "PilotMemoryRequests" is set to "200Mi"
    And Template value "IGCPULimit" is set to "1500m"
    And Template value "IGMemoryLimit" is set to "1200Mi"
    And Template value "IGCPURequests" is set to "80m"
    And Template value "IGMemoryRequests" is set to "200Mi"
    When Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    Then Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileSucceeded" of type "Ready" and status "True"
    And Istio CRDs "should" be present on cluster
    And Namespace "istio-system" has "namespaces.warden.kyma-project.io/validate" label and "istios.operator.kyma-project.io/managed-by-disclaimer" annotation
    And "Deployment" "istiod" in namespace "istio-system" is ready
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready
    And "pilot" has "limits" set to cpu - "1200m" and memory - "1200Mi"
    And "pilot" has "requests" set to cpu - "15m" and memory - "200Mi"
    And "ingress-gateway" has "limits" set to cpu - "1500m" and memory - "1200Mi"
    And "ingress-gateway" has "requests" set to cpu - "80m" and memory - "200Mi"

  Scenario: Additional Istio resources are present
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    When Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And "PeerAuthentication" "default" in namespace "istio-system" is "present"
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then "Istio CR" is not present on cluster
    And Namespace "istio-system" is "not present"

  Scenario: Installation of Istio module with Istio CR in different namespace
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "default"
    Then Istio CR "istio-sample" in namespace "default" has status "Error"
    And Istio CR "istio-sample" in namespace "default" has condition with reason "ReconcileFailed" of type "Ready" and status "False"
    And Istio CR "istio-sample" in namespace "default" has description "Stopped Istio CR reconciliation: Istio CR is not in kyma-system namespace"

  Scenario: Installation of Istio module with a second Istio CR in kyma-system namespace
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    When Istio CR "istio-sample-new" from "istio_cr_template" is applied in namespace "kyma-system"
    Then Istio CR "istio-sample-new" in namespace "kyma-system" has status "Warning"
    And Istio CR "istio-sample-new" in namespace "kyma-system" has condition with reason "OlderCRExists" of type "Ready" and status "False"
    And Istio CR "istio-sample-new" in namespace "kyma-system" has description "Stopped Istio CR reconciliation: only Istio CR istio-sample in kyma-system reconciles the module"

  Scenario: Istio module resources are reconciled, when they are deleted manually
    Given Istio CR "istio-sample" from "istio_cr_template" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CR "istio-sample" in namespace "kyma-system" has condition with reason "ReconcileSucceeded" of type "Ready" and status "True"
    And Namespace "istio-system" is "present"
    And "Deployment" "istiod" in namespace "istio-system" is deleted
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is deleted
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is deleted
    And Istio injection is "enabled" in namespace "default"
    And "PeerAuthentication" "default" in namespace "istio-system" is deleted
    And Httpbin application "test-app" deployment is created in namespace "default"
    # We need to update the Istio CR to trigger a reconciliation
    And Template value "ProxyCPURequest" is set to "79m"
    When Istio CR "istio-sample" from "istio_cr_template" is updated in namespace "kyma-system"
    Then "Deployment" "istiod" in namespace "istio-system" is ready
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready
    And "PeerAuthentication" "default" in namespace "istio-system" is "present"
    And Application pod "test-app" in namespace "default" has Istio proxy "present"
