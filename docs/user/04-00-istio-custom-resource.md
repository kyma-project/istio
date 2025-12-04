# Istio Custom Resource

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the kind and the format of data that Istio Controller uses to configure,
update, and manage the Istio installation. Applying the CR triggers the installation of Istio,
and deleting it triggers the uninstallation of Istio. The default Istio CR has the name `default`.

To get the up-to-date CRD in the `yaml` format, run the following command:

```bash
kubectl get crd istios.operator.kyma-project.io -o yaml
```
You are only allowed to use one Istio CR, which you must create in the `kyma-system` namespace.
If the namespace contains multiple Istio CRs, the oldest one reconciles the module.
Any additional Istio CR is placed in the `Warning` state.

## Sample Custom Resource
This is a sample Istio CR that configures Istio installation in your Kyma cluster.
    
```yaml
apiVersion: operator.kyma-project.io/v1alpha2
kind: Istio
metadata:
  name: default
  namespace: kyma-system
spec:
  config:
    gatewayExternalTrafficPolicy: Cluster
```

## Custom Resource Parameters
The following tables list all the possible parameters of a given resource together with their descriptions.

### APIVersions
- [operator.kyma-project.io/v1alpha2](#operatorkyma-projectiov1alpha2)

### Resource Types
- [Istio](#istio)

### Authorizer

Authorizer defines an external authorization provider configuration.
The defined authorizer can be referenced by name in an AuthorizationPolicy
with action CUSTOM to enforce requests to be authorized by the external authorization service.

Appears in:
- [Config](#config)

| Field | Description | Validation |
| --- | --- | --- |
| **name** <br /> string | A unique name identifying the extension authorization provider. | Required <br /> |
| **service** <br /> string | Specifies the service that implements the Envoy ext_authz HTTP authorization service.<br />The format is "[Namespace/]Hostname".<br />The specification of "Namespace" is required only when it is insufficient to unambiguously resolve a service in the service registry.<br />The "Hostname" is a fully qualified host name of a service defined by the Kubernetes service or ServiceEntry.<br />The recommended format is "[Namespace/]Hostname". | Optional |
| **port** <br /> integer | Specifies the port of the service. | Required <br /> |
| **headers** <br /> [Headers](#headers) | Specifies headers to be included, added or forwarded during authorization. | Optional |
| **pathPrefix** <br /> string | Specifies the prefix which will be included in the request sent to the authorization service.<br />The prefix might be constructed with special characters (e.g., "/test?original_path="). | Optional <br /> |
| **timeout** <br /> [Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta) | Specifies the timeout for the HTTP authorization request to the external service. | Optional <br /> |

### CniComponent

CniComponent defines configuration for CNI Istio component.

Appears in:
- [Components](#components)

| Field | Description | Validation |
| --- | --- | --- |
| **k8s** <br /> [CniK8sConfig](#cnik8sconfig) | CniK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec | Required <br /> |

### CniK8sConfig

Appears in:
- [CniComponent](#cnicomponent)

| Field | Description | Validation |
| --- | --- | --- |
| **affinity** <br /> [Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#affinity-v1-core) | Affinity defines the Pod scheduling affinity constraints: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity | Optional <br /> |
| **resources** <br /> [Resources](#resources) | Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ | Optional <br /> |

### Components

Appears in:
- [IstioSpec](#istiospec)

| Field | Description | Validation |
| --- | --- | --- |
| **pilot** <br /> [IstioComponent](#istiocomponent) | Pilot defines component configuration for Istiod | Optional |
| **ingressGateway** <br /> [IstioComponent](#istiocomponent) | IngressGateway defines component configurations for Istio Ingress Gateway | Optional |
| **cni** <br /> [CniComponent](#cnicomponent) | Cni defines component configuration for Istio CNI DaemonSet | Optional |
| **proxy** <br /> [ProxyComponent](#proxycomponent) | Proxy defines component configuration for Istio proxy sidecar | Optional |
| **egressGateway** <br /> [EgressGateway](#egressgateway) |  | Optional <br /> |

### ConditionReason

Underlying type: string

Appears in:
- [ReasonWithMessage](#reasonwithmessage)

| Field | Description |
| --- | --- |
| **ReconcileSucceeded** | Reconciliation finished with full success.<br /> |
| **ReconcileUnknown** | Reconciliation is in progress or failed previously.<br /> |
| **ReconcileRequeued** | Reconciliation is requeued to be tried again later.<br /> |
| **ReconcileFailed** | Reconciliation failed.<br /> |
| **ValidationFailed** | Reconciliation did not happen as validation of Istio Custom Resource failed.<br /> |
| **OlderCRExists** | Reconciliation did not happen as there exists an older Istio Custom Resource.<br /> |
| **OldestCRNotFound** | Reconciliation did not happen as the oldest Istio Custom Resource could not be found.<br /> |
| **IstioInstallNotNeeded** | Istio installtion is not needed.<br /> |
| **IstioInstallSucceeded** | Istio installation or uninstallation succeeded.<br /> |
| **IstioUninstallSucceeded** | Istio uninstallation succeeded.<br /> |
| **IstioInstallUninstallFailed** | Istio installation or uninstallation failed.<br /> |
| **IstioCustomResourceMisconfigured** | Istio Custom Resource has invalid configuration.<br /> |
| **IstioCustomResourcesDangling** | Istio Custom Resources are blocking Istio uninstallation.<br /> |
| **IstioVersionUpdateNotAllowed** | Istio version update is not allowed.<br /> |
| **CustomResourcesReconcileSucceeded** | Custom resources reconciliation succeeded.<br /> |
| **CustomResourcesReconcileFailed** | Custom resources reconciliation failed.<br /> |
| **ProxySidecarRestartSucceeded** | Proxy sidecar restart succeeded.<br /> |
| **ProxySidecarRestartFailed** | Proxy sidecar restart failed.<br /> |
| **ProxySidecarRestartPartiallySucceeded** | Proxy sidecar restart partially succeeded.<br /> |
| **ProxySidecarManualRestartRequired** | Proxy sidecar manual restart is required.<br /> |
| **IngressGatewayRestartSucceeded** | Istio ingress gateway restart succeeded.<br /> |
| **IngressGatewayRestartFailed** | Istio ingress gateway restart failed.<br /> |
| **EgressGatewayRestartSucceeded** | Istio egress gateway restart succeeded.<br /> |
| **EgressGatewayRestartFailed** | Istio egress gateway restart failed.<br /> |
| **IngressTargetingUserResourceFound** | Resource targeting Istio Ingress Gateway found.<br /> |
| **IngressTargetingUserResourceNotFound** | No resources targeting Istio Ingress Gateway found.<br /> |
| **IngressTargetingUserResourceDetectionFailed** | Resource targeting Istio Ingress Gateway detection failed.<br /> |


### Config

Config is the configuration for the Istio installation.

Appears in:
- [IstioSpec](#istiospec)

| Field | Description | Validation |
| --- | --- | --- |
| **numTrustedProxies** <br /> integer | Defines the number of trusted proxies deployed in front of the Istio gateway proxy. | Maximum: 4.294967295e+09 <br />Minimum: 0 <br /> |
| **forwardClientCertDetails** <br /> [XFCCStrategy](#xfccstrategy) | Defines the strategy of handling the **X-Forwarded-Client-Cert** header.<br />The default behavior is "SANITIZE". | Enum: [APPEND_FORWARD SANITIZE_SET SANITIZE ALWAYS_FORWARD_ONLY FORWARD_ONLY] <br />Optional <br /> |
| **authorizers** <br /> [Authorizer](#authorizer) array | Defines a list of external authorization providers. | Optional |
| **gatewayExternalTrafficPolicy** <br /> string | Defines the external traffic policy for the Istio Ingress Gateway Service. Valid configurations are "Local" or "Cluster". The external traffic policy set to "Local" preserves the client IP in the request, but also introduces the risk of unbalanced traffic distribution.<br />WARNING: Switching `externalTrafficPolicy` may result in a temporal increase in request delay. Make sure that this is acceptable. | Enum: [Local Cluster] <br />Optional <br /> |
| **telemetry** <br /> [Telemetry](#telemetry) | Defines the telemetry configuration of Istio. | Optional <br /> |

### EgressGateway

EgressGateway defines configuration for Istio egressGateway.

Appears in:
- [Components](#components)

| Field | Description | Validation |
| --- | --- | --- |
| **k8s** <br /> [KubernetesResourcesConfig](#kubernetesresourcesconfig) | Defines the Kubernetes resources configuration for Istio egress gateway. | Optional <br /> |
| **enabled** <br /> boolean | Enables or disables the Istio egress gateway. | Optional <br /> |

### Experimental

Appears in:
- [IstioSpec](#istiospec)

| Field | Description | Validation |
| --- | --- | --- |
| **pilot** <br /> [PilotFeatures](#pilotfeatures) |  | Optional |
| **enableDualStack** <br /> boolean | Enables dual-stack support. | Optional <br /> |
| **enableAmbient** <br /> boolean | Enables ambient mode support. | Optional <br /> |

### HPASpec

HPASpec defines configuration for HorizontalPodAutoscaler.

Appears in:
- [KubernetesResourcesConfig](#kubernetesresourcesconfig)

| Field | Description | Validation |
| --- | --- | --- |
| **maxReplicas** <br /> integer |  | Maximum: 2.147483647e+09 <br />Minimum: 0 <br /> |
| **minReplicas** <br /> integer |  | Maximum: 2.147483647e+09 <br />Minimum: 0 <br /> |

### Headers

Appears in:
- [Authorizer](#authorizer)

| Field | Description | Validation |
| --- | --- | --- |
| **inCheck** <br /> [InCheck](#incheck) | Defines headers to be included or added in check authorization request. | Optional |
| **toUpstream** <br /> [ToUpstream](#toupstream) | Defines headers to be forwarded to the upstream (to the backend service). | Optional |
| **toDownstream** <br /> [ToDownstream](#todownstream) | Defines headers to be forwarded to the downstream (the client). | Optional |

### InCheck

Appears in:
- [Headers](#headers)

| Field | Description | Validation |
| --- | --- | --- |
| **include** <br /> string array | List of client request headers that should be included in the authorization request sent to the authorization service.<br />Note that in addition to the headers specified here, the following headers are included by default:<br />1. *Host*, *Method*, *Path* and *Content-Length* are automatically sent.<br />2. *Content-Length* will be set to 0, and the request will not have a message body. However, the authorization request can include the buffered client request body (controlled by include_request_body_in_check setting), consequently the value of Content-Length of the authorization request reflects the size of its payload size. | Optional |
| **add** <br /> object (keys:string, values:string) | Set of additional fixed headers that should be included in the authorization request sent to the authorization service.<br />The Key is the header name and value is the header value.<br />Note that client request of the same key or headers specified in `Include` will be overridden. | Optional |

### Istio

Istio contains Istio CR specification and current status.

| Field | Description | Validation |
| --- | --- | --- |
| **apiVersion** <br /> string | `operator.kyma-project.io/v1alpha2` | Optional |
| **kind** <br /> string | `Istio` | Optional |
| **metadata** <br /> [ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta) | For more information on the metadata fields, see Kubernetes API documentation. | Optional |
| **spec** <br /> [IstioSpec](#istiospec) | Spec defines the desired state of the Istio installation. | Optional |
| **status** <br /> [IstioStatus](#istiostatus) | Status represents the current state of the Istio installation. | Optional |

### IstioComponent

IstioComponent defines configuration for generic Istio component (ingress gateway, istiod).

Appears in:
- [Components](#components)

| Field | Description | Validation |
| --- | --- | --- |
| **k8s** <br /> [KubernetesResourcesConfig](#kubernetesresourcesconfig) |  | Required <br /> |

### IstioSpec

IstioSpec describes the desired specification for installing or updating Istio.

Appears in:
- [Istio](#istio)

| Field | Description | Validation |
| --- | --- | --- |
| **config** <br /> [Config](#config) | Defines configuration of the Istio installation. | Optional <br /> |
| **components** <br /> [Components](#components) | Defines configuration of Istio components. | Optional <br /> |
| **experimental** <br /> [Experimental](#experimental) | Defines experimental configuration options. | Optional <br /> |
| **compatibilityMode** <br /> boolean | Enables compatibility mode for Istio installation. | Optional <br /> |

### IstioStatus

IstioStatus defines the observed state of IstioCR.

Appears in:
- [Istio](#istio)

| Field | Description | Validation |
| --- | --- | --- |
| **state** <br /> [State](#state) | State signifies the current state of CustomObject. Value<br />can be one of ("Ready", "Processing", "Error", "Deleting", "Warning"). | Enum: [Processing Deleting Ready Error Warning] <br />Required <br /> |
| **conditions** <br /> [Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#condition-v1-meta) |  Conditions associated with IstioStatus. | Optional |
| **description** <br /> string | Description of Istio status. | Optional |

### KubernetesResourcesConfig

KubernetesResourcesConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec

Appears in:
- [EgressGateway](#egressgateway)
- [IstioComponent](#istiocomponent)

| Field | Description | Validation |
| --- | --- | --- |
| **hpaSpec** <br /> [HPASpec](#hpaspec) | HPASpec defines configuration for HorizontalPodAutoscaler: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/ | Optional <br /> |
| **strategy** <br /> [Strategy](#strategy) | Strategy defines configuration for rolling updates: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment | Optional <br /> |
| **resources** <br /> [Resources](#resources) | Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ | Optional <br /> |

### Metrics

Appears in:
- [Telemetry](#telemetry)

| Field | Description | Validation |
| --- | --- | --- |
| **prometheusMerge** <br /> boolean | Defines whether the prometheusMerge feature is enabled. If yes, appropriate prometheus.io annotations will be added to all data plane pods to set up scraping.<br />If these annotations already exist, they will be overwritten. With this option, the Envoy sidecar will merge Istio’s metrics with the application metrics.<br />The merged metrics will be scraped from :15020/stats/prometheus. | Optional <br /> |

### PilotFeatures

Appears in:
- [Experimental](#experimental)

| Field | Description | Validation |
| --- | --- | --- |
| **enableAlphaGatewayAPI** <br /> boolean |  | Optional |
| **enableMultiNetworkDiscoverGatewayAPI** <br /> boolean |  | Optional |

### ProxyComponent

ProxyComponent defines configuration for Istio proxies.

Appears in:
- [Components](#components)

| Field | Description | Validation |
| --- | --- | --- |
| **k8s** <br /> [ProxyK8sConfig](#proxyk8sconfig) |  | Required <br /> |

### ProxyK8sConfig

ProxyK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec

Appears in:
- [ProxyComponent](#proxycomponent)

| Field | Description | Validation |
| --- | --- | --- |
| **resources** <br /> [Resources](#resources) |  | Optional |


### ResourceClaims

Appears in:
- [Resources](#resources)

| Field | Description | Validation |
| --- | --- | --- |
| **cpu** <br /> string |  | Pattern: `^([0-9]+m?\|[0-9]\.[0-9]\{1,3\})$` <br /> |
| **memory** <br /> string |  | Pattern: `^[0-9]+(((\.[0-9]+)?(E\|P\|T\|G\|M\|k\|Ei\|Pi\|Ti\|Gi\|Mi\|Ki\|m)?)\|(e[0-9]+))$` <br /> |

### Resources

Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

Appears in:
- [CniK8sConfig](#cnik8sconfig)
- [KubernetesResourcesConfig](#kubernetesresourcesconfig)
- [ProxyK8sConfig](#proxyk8sconfig)

| Field | Description | Validation |
| --- | --- | --- |
| **limits** <br /> [ResourceClaims](#resourceclaims) |  | Optional |
| **requests** <br /> [ResourceClaims](#resourceclaims) |  | Optional |

### RollingUpdate

RollingUpdate defines configuration for rolling updates: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment

Appears in:
- [Strategy](#strategy)

| Field | Description | Validation |
| --- | --- | --- |
| **maxSurge** <br /> [IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#intorstring-intstr-util) |  | Pattern: `^[0-9]+%?$` <br />XIntOrString: \{\} <br /> |
| **maxUnavailable** <br /> [IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#intorstring-intstr-util) |  | Pattern: `^((100\|[0-9]\{1,2\})%\|[0-9]+)$` <br />XIntOrString: \{\} <br /> |

### State

Underlying type: string

Appears in:
- [IstioStatus](#istiostatus)

| Field | Description |
| --- | --- |
| **Ready** | Ready is reported when the Istio installation / upgrade process has completed successfully.<br /> |
| **Processing** | Processing is reported when the Istio installation / upgrade process is in progress.<br /> |
| **Error** | Error is reported when the Istio installation / upgrade process has failed.<br /> |
| **Deleting** | Deleting is reported when the Istio installation / upgrade process is being deleted.<br /> |
| **Warning** | Warning is reported when the Istio installation / upgrade process has completed with warnings.<br />This state warrants user attention, as some features may not work as expected.<br /> |

### Strategy

Strategy defines rolling update strategy.

Appears in:
- [KubernetesResourcesConfig](#kubernetesresourcesconfig)

| Field | Description | Validation |
| --- | --- | --- |
| **rollingUpdate** <br /> [RollingUpdate](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#rollingupdatedeployment-v1-apps) |  | Required <br /> |

### Telemetry

Appears in:
- [Config](#config)

| Field | Description | Validation |
| --- | --- | --- |
| **metrics** <br /> [Metrics](#metrics) | Istio telemetry configuration related to metrics | Optional <br /> |

### ToDownstream

Appears in:
- [Headers](#headers)

| Field | Description | Validation |
| --- | --- | --- |
| **onAllow** <br /> string array | List of headers from the authorization service that should be forwarded to downstream when the authorization check result is allowed (HTTP code 200).<br />If not specified, the original response will not be modified and forwarded to downstream as-is.<br />Note, any existing headers will be overridden. | Optional |
| **onDeny** <br /> string array | List of headers from the authorization service that should be forwarded to downstream when the authorization check result is not allowed (HTTP code other than 200).<br />If not specified, all the authorization response headers, except *Authority (Host)* will be in the response to the downstream.<br />When a header is included in this list, *Path*, *Status*, *Content-Length*, *WWWAuthenticate* and *Location* are automatically added.<br />Note, the body from the authorization service is always included in the response to downstream. | Optional |

### ToUpstream

Appears in:
- [Headers](#headers)

| Field | Description | Validation |
| --- | --- | --- |
| **onAllow** <br /> string array | List of headers from the authorization service that should be added or overridden in the original request and forwarded to the upstream when the authorization check result is allowed (HTTP code 200).<br />If not specified, the original request will not be modified and forwarded to backend as-is.<br />Note, any existing headers will be overridden. | Optional |

### XFCCStrategy

Defines how to handle the x-forwarded-client-cert (XFCC) of the HTTP header.
XFCC is a proxy header that indicates certificate information of part or all of the clients or proxies that a request has passed through on its route from the client to the server.

Underlying type: string

Appears in:
- [Config](#config)

| Field | Description |
| --- | --- |
| **APPEND_FORWARD** | When the client connection is mutual TLS (mTLS), append the client certificate information to the request’s XFCC header and forward it.<br /> |
| **SANITIZE_SET** | When the client connection is mTLS, reset the XFCC header with the client certificate information and send it to the next hop.<br /> |
| **SANITIZE** | Do not send the XFCC header to the next hop.<br /> |
| **ALWAYS_FORWARD_ONLY** | Always forward the XFCC header in the request, regardless of whether the client connection is mTLS.<br /> |
| **FORWARD_ONLY** | When the client connection is mTLS, forward the XFCC header in the request.<br /> |

