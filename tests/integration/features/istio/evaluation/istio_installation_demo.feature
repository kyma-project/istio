Feature: Installing Istio module with evaluation profile
  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Evaluation"
    And Istio CRD is installed
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario:
    When Istio CR "istio-sample" is applied in namespace "default"
    Then Istio CR "istio-sample" in namespace "default" has status "Ready"
    Then "proxy" has "requests" set to cpu - "10m" and memory - "32Mi"
    And "proxy" has "limits" set to cpu - "250m" and memory - "254Mi"
    And "ingress-gateway" has "requests" set to cpu - "10m" and memory - "32Mi"
    And "ingress-gateway" has "limits" set to cpu - "1000m" and memory - "1024Mi"
    And "proxy_init" has "requests" set to cpu - "10m" and memory - "10Mi"
    And "proxy_init" has "limits" set to cpu - "100m" and memory - "50Mi"
    And "pilot" has "requests" set to cpu - "50m" and memory - "128Mi"
    And "pilot" has "limits" set to cpu - "1000m" and memory - "1024Mi"
    And "egress-gateway" has "requests" set to cpu - "10m" and memory - "120Mi"
    And "egress-gateway" has "limits" set to cpu - "2000m" and memory - "1024Mi"
