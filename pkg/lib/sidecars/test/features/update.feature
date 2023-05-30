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

  Scenario: Updating CPU requests Proxy configuration
    Given there is a cluster with Istio "1.14.4", default injection == "true" and CNI enabled == "true"
    And there are Pods with Istio "1.14.4" sidecar and resource "requests" cpu "10m" and memory "100Mi"
    When a restart happens with target Istio "1.14.4" and sidecar resource "requests" set to cpu "20m" and memory "100Mi"
    Then no resource that is not supposed to be restarted is restarted
    And no resource that is not supposed to be deleted is deleted
    And all required resources are restarted
    And all required resources are deleted

  Scenario: Updating memory requests Proxy configuration
    Given there is a cluster with Istio "1.14.4", default injection == "true" and CNI enabled == "true"
    And there are Pods with Istio "1.14.4" sidecar and resource "requests" cpu "10m" and memory "100Mi"
    When a restart happens with with target Istio "1.14.4" and sidecar resource "requests" set to cpu "10m" and memory "200Mi"
    Then no resource that is not supposed to be restarted is restarted
    And no resource that is not supposed to be deleted is deleted
    And all required resources are restarted
    And all required resources are deleted

  Scenario: Updating cpu limits Proxy configuration
    Given there is a cluster with Istio "1.14.4", default injection == "true" and CNI enabled == "true"
    And there are Pods with Istio "1.14.4" sidecar and resource "limits" cpu "100m" and memory "200Mi"
    When a restart happens with target Istio "1.14.4" and sidecar resource "limits" set to cpu "200m" and memory "200Mi"
    Then no resource that is not supposed to be restarted is restarted
    And no resource that is not supposed to be deleted is deleted
    And all required resources are restarted
    And all required resources are deleted

  Scenario: Updating cpu limits Proxy configuration
    Given there is a cluster with Istio "1.14.4", default injection == "true" and CNI enabled == "true"
    And there are Pods with Istio "1.14.4" sidecar and resource "limits" cpu "100m" and memory "200Mi"
    When a restart happens with target Istio "1.14.4" and sidecar resource "limits" set to cpu "100m" and memory "300Mi"
    Then no resource that is not supposed to be restarted is restarted
    And no resource that is not supposed to be deleted is deleted
    And all required resources are restarted
    And all required resources are deleted