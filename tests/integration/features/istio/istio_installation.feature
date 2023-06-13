Feature: Installing and uninstalling Istio module

  Background:
    Given "Istio CR" is not present on cluster
    And Istio CRD is installed
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Installation of Istio module
    Given Template value "PilotCPULimit" is set to "1200m"
    And Template value "PilotMemoryLimit" is set to "1200Mi"
    And Template value "PilotCPURequests" is set to "15m"
    And Template value "PilotMemoryRequests" is set to "200Mi"
    And Template value "IGCPULimit" is set to "1500m"
    And Template value "IGMemoryLimit" is set to "1200Mi"
    And Template value "IGCPURequests" is set to "80m"
    And Template value "IGMemoryRequests" is set to "200Mi"
    When Istio CR "istio-sample" is applied in namespace "default"
    Then Istio CR "istio-sample" in namespace "default" has status "Ready"
    And Istio CRDs "should" be present on cluster
    And Namespace "istio-system" has "namespaces.warden.kyma-project.io/validate" label and "istios.operator.kyma-project.io/managed-by-disclaimer" annotation
    And "Deployment" "istiod" in namespace "istio-system" is ready
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready
    And "pilot" has "limits" set to cpu - "1200m" and memory - "1200Mi"
    And "pilot" has "requests" set to cpu - "15m" and memory - "200Mi"
    And "ingress-gateway" has "limits" set to cpu - "1500m" and memory - "1200Mi"
    And "ingress-gateway" has "requests" set to cpu - "80m" and memory - "200Mi"

  Scenario: Uninstallation of Istio module
    Given Istio CR "istio-sample" is applied in namespace "default"
    And Istio CR "istio-sample" in namespace "default" has status "Ready"
    And Namespace "istio-system" is "present"
    And Istio injection is enabled in namespace "default"
    And Application "test-app" is running in namespace "default"
    And Application pod "test-app" in namespace "default" has Istio proxy "present"
    When "Istio CR" "istio-sample" in namespace "default" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"
    And Application pod "test-app" in namespace "default" has Istio proxy "not present"

  Scenario: Uninstallation respects the Istio resources created by the user
    Given Istio CR "istio-sample" is applied in namespace "default"
    And Istio CR "istio-sample" in namespace "default" has status "Ready"
    And Namespace "istio-system" is "present"
    And Destination rule "customer-destination-rule" in namespace "default" with host "testing-svc.default.svc.cluster.local" exists
    When "Istio CR" "istio-sample" in namespace "default" is deleted
    Then Istio CR "istio-sample" in namespace "default" has status "Error"
    And Istio CRDs "should" be present on cluster
    And Namespace "istio-system" is "present"
    When "DestinationRule" "customer-destination-rule" in namespace "default" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"

Scenario: Uninstallation of Istio module if Istio was manually deleted
    Given Istio CR "istio-sample" is applied in namespace "default"
    And Istio CR "istio-sample" in namespace "default" has status "Ready"
    And Namespace "istio-system" is "present"
    And Istio is manually uninstalled
    When "Istio CR" "istio-sample" in namespace "default" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"
