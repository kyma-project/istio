Feature: Installing and uninstalling Istio module

  Background:
    Given "Istio CR" is not present on cluster
    And Evaluated cluster size is "Production"
    And Istio CRD is installed
    And Namespace "istio-system" is "not present"
    And "Deployment" "istio-controller-manager" in namespace "kyma-system" is ready

  Scenario: Installation of Istio module with default values
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    Then Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio "istio-ingressgateway" service has annotation "dns.gardener.cloud/dnsnames" on "Gardener" cluster
    And "proxy" has "requests" set to cpu - "10m" and memory - "192Mi"
    And "proxy" has "limits" set to cpu - "1000m" and memory - "1024Mi"
    And "ingress-gateway" has "requests" set to cpu - "100m" and memory - "128Mi"
    And "ingress-gateway" has "limits" set to cpu - "2000m" and memory - "1024Mi"
    And "proxy_init" has "requests" set to cpu - "10m" and memory - "10Mi"
    And "proxy_init" has "limits" set to cpu - "100m" and memory - "50Mi"
    And "pilot" has "requests" set to cpu - "100m" and memory - "512Mi"
    And "pilot" has "limits" set to cpu - "4000m" and memory - "2Gi"
    And "egress-gateway" has "requests" set to cpu - "10m" and memory - "120Mi"
    And "egress-gateway" has "limits" set to cpu - "2000m" and memory - "1024Mi"

  Scenario: Installation of Istio module
    Given Template value "PilotCPULimit" is set to "1200m"
    And Template value "PilotMemoryLimit" is set to "1200Mi"
    And Template value "PilotCPURequests" is set to "15m"
    And Template value "PilotMemoryRequests" is set to "200Mi"
    And Template value "IGCPULimit" is set to "1500m"
    And Template value "IGMemoryLimit" is set to "1200Mi"
    And Template value "IGCPURequests" is set to "80m"
    And Template value "IGMemoryRequests" is set to "200Mi"
    When Istio CR "istio-sample" is applied in namespace "kyma-system"
    Then Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Istio CRDs "should" be present on cluster
    And Namespace "istio-system" has "namespaces.warden.kyma-project.io/validate" label and "istios.operator.kyma-project.io/managed-by-disclaimer" annotation
    And "Deployment" "istiod" in namespace "istio-system" is ready
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And Istio "istio-ingressgateway" service has annotation "dns.gardener.cloud/dnsnames" on "Gardener" cluster
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready
    And "pilot" has "limits" set to cpu - "1200m" and memory - "1200Mi"
    And "pilot" has "requests" set to cpu - "15m" and memory - "200Mi"
    And "ingress-gateway" has "limits" set to cpu - "1500m" and memory - "1200Mi"
    And "ingress-gateway" has "requests" set to cpu - "80m" and memory - "200Mi"

  Scenario: Uninstallation of Istio module
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "istio-system" is "present"
    And Istio injection is "enabled" in namespace "default"
    And Application "test-app" is running in namespace "default"
    And Application pod "test-app" in namespace "default" has Istio proxy "present"
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"
    And Application pod "test-app" in namespace "default" has Istio proxy "not present"

  Scenario: Uninstallation respects the Istio resources created by the user
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "istio-system" is "present"
    And Destination rule "customer-destination-rule" in namespace "default" with host "testing-svc.default.svc.cluster.local" exists
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then Istio CR "istio-sample" in namespace "kyma-system" has status "Warning"
    And Istio CR "istio-sample" in namespace "kyma-system" has description "There are Istio resources that block deletion. Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning"
    And Istio CRDs "should" be present on cluster
    And Namespace "istio-system" is "present"
    When "DestinationRule" "customer-destination-rule" in namespace "default" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"

  Scenario: Uninstallation of Istio module if Istio was manually deleted
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "istio-system" is "present"
    And Istio is manually uninstalled
    And Namespace "istio-system" is "not present"
    And Istio CRDs "should not" be present on cluster
    When "Istio CR" "istio-sample" in namespace "kyma-system" is deleted
    Then "Istio CR" is not present on cluster
    And Istio CRDs "should not" be present on cluster
    And Namespace "istio-system" is "not present"

  Scenario: Installation of Istio module with Istio CR in different namespace
    Given Istio CR "istio-sample" is applied in namespace "default"
    Then Istio CR "istio-sample" in namespace "default" has status "Error"
    And Istio CR "istio-sample" in namespace "default" has description "Stopped Istio CR reconciliation: istio CR is not in kyma-system namespace"

  Scenario: Installation of Istio module with a second Istio CR in kyma-system namespace
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    When Istio CR "istio-sample-new" is applied in namespace "kyma-system"
    Then Istio CR "istio-sample-new" in namespace "kyma-system" has status "Error"
    And Istio CR "istio-sample-new" in namespace "kyma-system" has description "Stopped Istio CR reconciliation: only Istio CR istio-sample in kyma-system reconciles the module"

  Scenario: Istio module resources are reconciled, when they are deleted manually
    Given Istio CR "istio-sample" is applied in namespace "kyma-system"
    And Istio CR "istio-sample" in namespace "kyma-system" has status "Ready"
    And Namespace "istio-system" is "present"
    And "Deployment" "istiod" in namespace "istio-system" is deleted
    And "istiooperator" "installed-state-default-operator" in namespace "istio-system" is deleted
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is deleted
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is deleted
    And Istio injection is "enabled" in namespace "default"
    And "Gateway" "kyma-gateway" in namespace "kyma-system" is deleted
    And "EnvoyFilter" "kyma-referer" in namespace "istio-system" is deleted
    And "PeerAuthentication" "default" in namespace "istio-system" is deleted
    And "VirtualService" "istio-healthz" in namespace "istio-system" is deleted
    And "ConfigMap" "istio-control-plane-grafana-dashboard" in namespace "kyma-system" is deleted
    And "ConfigMap" "istio-mesh-grafana-dashboard" in namespace "kyma-system" is deleted
    And "ConfigMap" "istio-performance-grafana-dashboard" in namespace "kyma-system" is deleted
    And "ConfigMap" "istio-service-grafana-dashboard" in namespace "kyma-system" is deleted
    And "ConfigMap" "istio-workload-grafana-dashboard" in namespace "kyma-system" is deleted
    And Application "test-app" is running in namespace "default"
    # We need to update the Istio CR to trigger a reconciliation
    And Template value "ProxyCPURequest" is set to "79m"
    When Istio CR "istio-sample" is updated in namespace "kyma-system"
    Then "Deployment" "istiod" in namespace "istio-system" is ready
    And "istiooperator" "installed-state-default-operator" in namespace "istio-system" is "present"
    And "Deployment" "istio-ingressgateway" in namespace "istio-system" is ready
    And "DaemonSet" "istio-cni-node" in namespace "istio-system" is ready
    And "Gateway" "kyma-gateway" in namespace "kyma-system" is "present"
    And "EnvoyFilter" "kyma-referer" in namespace "istio-system" is "present"
    And "PeerAuthentication" "default" in namespace "istio-system" is "present"
    And "VirtualService" "istio-healthz" in namespace "istio-system" is "present"
    And "ConfigMap" "istio-control-plane-grafana-dashboard" in namespace "kyma-system" is "present"
    And "ConfigMap" "istio-mesh-grafana-dashboard" in namespace "kyma-system" is "present"
    And "ConfigMap" "istio-performance-grafana-dashboard" in namespace "kyma-system" is "present"
    And "ConfigMap" "istio-service-grafana-dashboard" in namespace "kyma-system" is "present"
    And "ConfigMap" "istio-workload-grafana-dashboard" in namespace "kyma-system" is "present"
    And Application pod "test-app" in namespace "default" has Istio proxy "present"