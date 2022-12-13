Feature: Istio upgrade

  Scenario: Upgrading Istio version
    Given there is cluster with Istio "1.14.4"
    And there are pods with not yet injected sidecars
    When a restart happens with default injection == "false" and CNI enabled == "false"
    Then all required resources are restarted
    And all required resources are deleted