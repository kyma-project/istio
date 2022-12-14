Feature: Istio upgrade

  Scenario: Upgrading Istio version
    Given there is a cluster with Istio "1.14.4", default injection == "false" and CNI enabled == "false"
    And there are Pods missing sidecar
    And there are Pods with Istio "1.13.3" sidecar
    And there are Pods with Istio "1.14.4" sidecar
    And there are not ready Pods
    When a restart happens with target Istio "1.14.4"
    Then no unrequired resource is restarted
    And no unrequired resource is deleted
    And all required resources are restarted
    And all required resources are deleted