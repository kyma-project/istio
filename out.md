# API Reference

## Packages
- [operator.kyma-project.io/v1alpha2](#operatorkyma-projectiov1alpha2)


## operator.kyma-project.io/v1alpha2

Package v1alpha2 contains API Schema definitions for the operator v1alpha2 API group

### Resource Types
- [Istio](#istio)
- [IstioList](#istiolist)



#### Authorizer







_Appears in:_
- [Config](#config)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | A unique name identifying the extension authorization provider. |  | Required: \{\} <br /> |
| `service` _string_ | Specifies the service that implements the Envoy ext_authz HTTP authorization service.<br />The format is "\[\<Namespace>/\]\<Hostname\>".<br />The specification of "\<Namespace\>"<br />is required only when it is insufficient to unambiguously resolve a service in the service registry.<br />The "\<Hostname\>" is a fully qualified host name of a service defined by the Kubernetes service or ServiceEntry.<br />The recommended format is "\[\<Namespace\>\/\]\<Hostname\>"<br />Example: "my-ext-authz.foo.svc.cluster.local" or "bar/my-ext-authz". |  | Required: \{\} <br /> |
| `port` _integer_ | Specifies the port of the service. |  | Required: \{\} <br /> |
| `headers` _[Headers](#headers)_ | Specifies headers to be included, added or forwarded during authorization. |  |  |
| `pathPrefix` _string_ | Specifies the prefix which will be included in the request sent to the authorization service.<br />The prefix might be constructed with special characters (e.g., "/test?original_path="). |  | Optional: \{\} <br /> |
| `timeout` _[Duration](#duration)_ | Specifies the timeout for the HTTP authorization request to the external service.<br />Default timeout, as defined in [Envoy](https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/ext_authz/v3/ext_authz.proto#extensions-filters-http-ext-authz-v3-extauthz) is 200ms. |  | Optional: \{\} <br /> |


#### CniComponent



CniComponent defines configuration for CNI Istio component.



_Appears in:_
- [Components](#components)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `k8s` _[CniK8sConfig](#cnik8sconfig)_ | CniK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec |  | Required: \{\} <br /> |


#### CniK8sConfig







_Appears in:_
- [CniComponent](#cnicomponent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ | Affinity defines the Pod scheduling affinity constraints: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity |  | Optional: \{\} <br /> |
| `resources` _[Resources](#resources)_ | Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |  | Optional: \{\} <br /> |


#### Components







_Appears in:_
- [IstioSpec](#istiospec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `pilot` _[IstioComponent](#istiocomponent)_ | Pilot defines component configuration for Istiod |  |  |
| `ingressGateway` _[IstioComponent](#istiocomponent)_ | IngressGateway defines component configurations for Istio Ingress Gateway |  |  |
| `cni` _[CniComponent](#cnicomponent)_ | Cni defines component configuration for Istio CNI DaemonSet |  |  |
| `proxy` _[ProxyComponent](#proxycomponent)_ | Proxy defines component configuration for Istio proxy sidecar |  |  |
| `egressGateway` _[EgressGateway](#egressgateway)_ |  |  | Optional: \{\} <br /> |


#### ConditionReason

_Underlying type:_ _string_





_Appears in:_
- [ReasonWithMessage](#reasonwithmessage)

| Field | Description |
| --- | --- |
| `ReconcileSucceeded` | Reconciliation finished with full success.<br /> |
| `ReconcileUnknown` | Reconciliation is in progress or failed previously.<br /> |
| `ReconcileRequeued` | Reconciliation is requeued to be tried again later.<br /> |
| `ReconcileFailed` | Reconciliation failed.<br /> |
| `ValidationFailed` | Reconciliation did not happen as validation of Istio Custom Resource failed.<br /> |
| `OlderCRExists` | Reconciliation did not happen as there exists an older Istio Custom Resource.<br /> |
| `OldestCRNotFound` | Reconciliation did not happen as the oldest Istio Custom Resource could not be found.<br /> |
| `IstioInstallNotNeeded` | Istio installtion is not needed.<br /> |
| `IstioInstallSucceeded` | Istio installation or uninstallation succeeded.<br /> |
| `IstioUninstallSucceeded` | Istio uninstallation succeeded.<br /> |
| `IstioInstallUninstallFailed` | Istio installation or uninstallation failed.<br /> |
| `IstioCustomResourceMisconfigured` | Istio Custom Resource has invalid configuration.<br /> |
| `IstioCustomResourcesDangling` | Istio Custom Resources are blocking Istio uninstallation.<br /> |
| `IstioVersionUpdateNotAllowed` | Istio version update is not allowed.<br /> |
| `CustomResourcesReconcileSucceeded` | Custom resources reconciliation succeeded.<br /> |
| `CustomResourcesReconcileFailed` | Custom resources reconciliation failed.<br /> |
| `ProxySidecarRestartSucceeded` | Proxy sidecar restart succeeded.<br /> |
| `ProxySidecarRestartFailed` | Proxy sidecar restart failed.<br /> |
| `ProxySidecarRestartPartiallySucceeded` | Proxy sidecar restart partially succeeded.<br /> |
| `ProxySidecarManualRestartRequired` | Proxy sidecar manual restart is required.<br /> |
| `IngressGatewayRestartSucceeded` | Istio ingress gateway restart succeeded.<br /> |
| `IngressGatewayRestartFailed` | Istio ingress gateway restart failed.<br /> |
| `EgressGatewayRestartSucceeded` | Istio egress gateway restart succeeded.<br /> |
| `EgressGatewayRestartFailed` | Istio egress gateway restart failed.<br /> |
| `IngressTargetingUserResourceFound` | Resource targeting Istio Ingress Gateway found.<br /> |
| `IngressTargetingUserResourceNotFound` | No resources targeting Istio Ingress Gateway found.<br /> |
| `IngressTargetingUserResourceDetectionFailed` | Resource targeting Istio Ingress Gateway detection failed.<br /> |




#### Config



Config is the configuration for the Istio installation.



_Appears in:_
- [IstioSpec](#istiospec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `numTrustedProxies` _integer_ | Defines the number of trusted proxies deployed in front of the Istio gateway proxy. |  | Maximum: 4.294967295e+09 <br />Minimum: 0 <br /> |
| `authorizers` _[Authorizer](#authorizer) array_ | Defines a list of external authorization providers. |  |  |
| `gatewayExternalTrafficPolicy` _string_ | Defines the external traffic policy for the Istio Ingress Gateway Service. Valid configurations are "Local" or "Cluster". The external traffic policy set to "Local" preserves the client IP in the request, but also introduces the risk of unbalanced traffic distribution.<br />WARNING: Switching `externalTrafficPolicy` may result in a temporal increase in request delay. Make sure that this is acceptable. |  | Enum: [Local Cluster] <br />Optional: \{\} <br /> |
| `telemetry` _[Telemetry](#telemetry)_ | Defines the telemetry configuration of Istio. |  | Optional: \{\} <br /> |


#### EgressGateway



EgressGateway defines configuration for Istio egressGateway.



_Appears in:_
- [Components](#components)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `k8s` _[KubernetesResourcesConfig](#kubernetesresourcesconfig)_ | Defines the Kubernetes resources configuration for Istio egress gateway. |  | Optional: \{\} <br /> |
| `enabled` _boolean_ | Enables or disables the Istio egress gateway. |  | Optional: \{\} <br /> |


#### Experimental







_Appears in:_
- [IstioSpec](#istiospec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `pilot` _[PilotFeatures](#pilotfeatures)_ |  |  |  |


#### HPASpec



HPASpec defines configuration for HorizontalPodAutoscaler.



_Appears in:_
- [KubernetesResourcesConfig](#kubernetesresourcesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxReplicas` _integer_ |  |  | Maximum: 2.147483647e+09 <br />Minimum: 0 <br /> |
| `minReplicas` _integer_ |  |  | Maximum: 2.147483647e+09 <br />Minimum: 0 <br /> |


#### Headers







_Appears in:_
- [Authorizer](#authorizer)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `inCheck` _[InCheck](#incheck)_ | Defines headers to be included or added in check authorization request. |  |  |
| `toUpstream` _[ToUpstream](#toupstream)_ | Defines headers to be forwarded to the upstream (to the backend service). |  |  |
| `toDownstream` _[ToDownstream](#todownstream)_ | Defines headers to be forwarded to the downstream (the client). |  |  |


#### InCheck







_Appears in:_
- [Headers](#headers)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `include` _string array_ | List of client request headers that should be included in the authorization request sent to the authorization service.<br />Note that in addition to the headers specified here, the following headers are included by default:<br />1. *Host*, *Method*, *Path* and *Content-Length* are automatically sent.<br />2. *Content-Length* will be set to 0, and the request will not have a message body. However, the authorization request can include the buffered client request body (controlled by include_request_body_in_check setting), consequently the value of Content-Length of the authorization request reflects the size of its payload size. |  |  |
| `add` _object (keys:string, values:string)_ | Set of additional fixed headers that should be included in the authorization request sent to the authorization service.<br />The Key is the header name and value is the header value.<br />Note that client request of the same key or headers specified in `Include` will be overridden. |  |  |


#### Istio



Istio contains Istio CR specification and current status.



_Appears in:_
- [IstioList](#istiolist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `operator.kyma-project.io/v1alpha2` | | |
| `kind` _string_ | `Istio` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[IstioSpec](#istiospec)_ | Spec defines the desired state of the Istio installation. |  |  |


#### IstioComponent



IstioComponent defines configuration for generic Istio component (ingress gateway, istiod).



_Appears in:_
- [Components](#components)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `k8s` _[KubernetesResourcesConfig](#kubernetesresourcesconfig)_ |  |  | Required: \{\} <br /> |


#### IstioList



IstioList contains a list of Istio's.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `operator.kyma-project.io/v1alpha2` | | |
| `kind` _string_ | `IstioList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Istio](#istio) array_ |  |  |  |


#### IstioSpec



IstioSpec describes the desired specification for installing or updating Istio.



_Appears in:_
- [Istio](#istio)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `config` _[Config](#config)_ | Defines configuration of the Istio installation. |  | Optional: \{\} <br /> |
| `components` _[Components](#components)_ | Defines configuration of Istio components. |  | Optional: \{\} <br /> |
| `experimental` _[Experimental](#experimental)_ | Defines experimental configuration options. |  | Optional: \{\} <br /> |
| `compatibilityMode` _boolean_ | Enables compatibility mode for Istio installation. |  | Optional: \{\} <br /> |




#### KubernetesResourcesConfig



KubernetesResourcesConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec



_Appears in:_
- [EgressGateway](#egressgateway)
- [IstioComponent](#istiocomponent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `hpaSpec` _[HPASpec](#hpaspec)_ | HPASpec defines configuration for HorizontalPodAutoscaler: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/ |  | Optional: \{\} <br /> |
| `strategy` _[Strategy](#strategy)_ | Strategy defines configuration for rolling updates: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment |  | Optional: \{\} <br /> |
| `resources` _[Resources](#resources)_ | Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |  | Optional: \{\} <br /> |


#### Metrics







_Appears in:_
- [Telemetry](#telemetry)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `prometheusMerge` _boolean_ | Defines whether the prometheusMerge feature is enabled. If yes, appropriate prometheus.io annotations will be added to all data plane pods to set up scraping.<br />If these annotations already exist, they will be overwritten. With this option, the Envoy sidecar will merge Istio’s metrics with the application metrics.<br />The merged metrics will be scraped from :15020/stats/prometheus. |  | Optional: \{\} <br /> |


#### PilotFeatures







_Appears in:_
- [Experimental](#experimental)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enableAlphaGatewayAPI` _boolean_ |  |  |  |
| `enableMultiNetworkDiscoverGatewayAPI` _boolean_ |  |  |  |


#### ProxyComponent



ProxyComponent defines configuration for Istio proxies.



_Appears in:_
- [Components](#components)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `k8s` _[ProxyK8sConfig](#proxyk8sconfig)_ |  |  | Required: \{\} <br /> |


#### ProxyK8sConfig



ProxyK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec



_Appears in:_
- [ProxyComponent](#proxycomponent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[Resources](#resources)_ |  |  |  |




#### ResourceClaims







_Appears in:_
- [Resources](#resources)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cpu` _string_ |  |  | Pattern: `^([0-9]+m?\|[0-9]\.[0-9]\{1,3\})$` <br /> |
| `memory` _string_ |  |  | Pattern: `^[0-9]+(((\.[0-9]+)?(E\|P\|T\|G\|M\|k\|Ei\|Pi\|Ti\|Gi\|Mi\|Ki\|m)?)\|(e[0-9]+))$` <br /> |


#### Resources



Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/



_Appears in:_
- [CniK8sConfig](#cnik8sconfig)
- [KubernetesResourcesConfig](#kubernetesresourcesconfig)
- [ProxyK8sConfig](#proxyk8sconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `limits` _[ResourceClaims](#resourceclaims)_ |  |  |  |
| `requests` _[ResourceClaims](#resourceclaims)_ |  |  |  |


#### RollingUpdate



RollingUpdate defines configuration for rolling updates: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment



_Appears in:_
- [Strategy](#strategy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxSurge` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#intorstring-intstr-util)_ |  |  | Pattern: `^[0-9]+%?$` <br />XIntOrString: \{\} <br /> |
| `maxUnavailable` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#intorstring-intstr-util)_ |  |  | Pattern: `^((100\|[0-9]\{1,2\})%\|[0-9]+)$` <br />XIntOrString: \{\} <br /> |


#### State

_Underlying type:_ _string_





_Appears in:_
- [IstioStatus](#istiostatus)

| Field | Description |
| --- | --- |
| `Ready` | Ready is reported when the Istio installation / upgrade process has completed successfully.<br /> |
| `Processing` | Processing is reported when the Istio installation / upgrade process is in progress.<br /> |
| `Error` | Error is reported when the Istio installation / upgrade process has failed.<br /> |
| `Deleting` | Deleting is reported when the Istio installation / upgrade process is being deleted.<br /> |
| `Warning` | Warning is reported when the Istio installation / upgrade process has completed with warnings.<br />This state warrants user attention, as some features may not work as expected.<br /> |


#### Strategy



Strategy defines rolling update strategy.



_Appears in:_
- [KubernetesResourcesConfig](#kubernetesresourcesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `rollingUpdate` _[RollingUpdate](#rollingupdate)_ |  |  | Required: \{\} <br /> |


#### Telemetry







_Appears in:_
- [Config](#config)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metrics` _[Metrics](#metrics)_ | Istio telemetry configuration related to metrics |  | Optional: \{\} <br /> |


#### ToDownstream







_Appears in:_
- [Headers](#headers)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `onAllow` _string array_ | List of headers from the authorization service that should be forwarded to downstream when the authorization check result is allowed (HTTP code 200).<br />If not specified, the original response will not be modified and forwarded to downstream as-is.<br />Note, any existing headers will be overridden. |  |  |
| `onDeny` _string array_ | List of headers from the authorization service that should be forwarded to downstream when the authorization check result is not allowed (HTTP code other than 200).<br />If not specified, all the authorization response headers, except *Authority (Host)* will be in the response to the downstream.<br />When a header is included in this list, *Path*, *Status*, *Content-Length*, *WWWAuthenticate* and *Location* are automatically added.<br />Note, the body from the authorization service is always included in the response to downstream. |  |  |


#### ToUpstream







_Appears in:_
- [Headers](#headers)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `onAllow` _string array_ | List of headers from the authorization service that should be added or overridden in the original request and forwarded to the upstream when the authorization check result is allowed (HTTP code 200).<br />If not specified, the original request will not be modified and forwarded to backend as-is.<br />Note, any existing headers will be overridden. |  |  |


