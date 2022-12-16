Feature: Istio upgrade

  Scenario: Upgrading Istio version
    Given there is a cluster with Istio "1.14.4", default injection == "false" and CNI enabled == "false"
    And there are Pods with Istio "1.13.3" sidecar
    When a restart happens with target Istio "1.14.4"
    Then no resource that is not supposed to be restarted is restarted
    And no resource that is not supposed to be deleted is deleted
    And all required resources are restarted
    And all required resources are deleted

  Scenario: Pods missing sidecars
    Given there is a cluster with Istio "1.14.4", default injection == "false" and CNI enabled == "false"
    And there are Pods missing sidecar
    And there are not ready Pods
    When a restart happens with target Istio "1.14.4"
    Then no resource that is not supposed to be restarted is restarted
    And no resource that is not supposed to be deleted is deleted
    And all required resources are restarted
    And all required resources are deleted
  
  Scenario: Standard reconciliation
    Given there is a cluster with Istio "1.14.4", default injection == "false" and CNI enabled == "false"
    And there are Pods with Istio "1.14.4" sidecar
    When a restart happens with target Istio "1.14.4"
    Then no resource that is not supposed to be restarted is restarted
    And no resource that is not supposed to be deleted is deleted
    And all required resources are restarted
    And all required resources are deleted