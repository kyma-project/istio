# Istio custom resource

The `istios.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the kind and the format of data that Istio Controller uses to configure, update, and manage the Istio installation. Applying the CR triggers the installation of Istio, and deleting it triggers the uninstallation of Istio. To get the up-to-date CRD in the `yaml` format, run the following command:

```shell
kubectl get crd istios.operator.kyma-project.io -o yaml
```

## Specification

This table lists all the possible parameters of the given resource together with their descriptions:


**Spec:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **components.cni**  | object | Defines component configuration for Istio CNI DaemonSet. |
| **components.cni.k8s.affinity**  | object | Affinity is a group of affinity scheduling rules. To learn more, read about affininty in the [Istio documentation](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#Affinity).|
| **components.cni.k8s.resources**  | object | Defines [Kubernetes resources configuration](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). For more information, read about Resources in the [Istio documentation](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#Resources ). |
| **components.ingressGateway**  | object | Defines component configurations for Istio Ingress Gateway. |
| **components.ingressGateway.k8s.hpaSpec**  | object | Defines configuration for HorizontalPodAutoscaler. |
| **components.ingressGateway.k8s.hpaSpec.maxReplicas**  | integer | Specifies the upper limit for the number of Pods that can be set by the autoscaler. It cannot be smaller than **MinReplicas**. |
| **components.ingressGateway.k8s.hpaSpec.minReplicas**  | integer | Specifies the lower limit for the number of replicas to which the autoscaler can scale down. By default, it is set to 1 Pod. The value can be set to 0 if the alpha feature gate `HPAScaleToZero` is enabled and at least one Object or External metric is configured. Scaling is active as long as at least one metric value is available. |
| **components.ingressGateway.k8s.resources**  | object | Defines [Kubernetes resources configuration](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). To learn more, read the [Istio documentation](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#Resources). |
| **components.ingressGateway.k8s.strategy**  | object | Defines the rolling update strategy. To learn more, read about DeploymentStrategy in the [Istio documentation](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#DeploymentStrategy). |
| **components.pilot**  | object | Defines component configuration for Istiod. |
| **components.pilot.k8s.hpaSpec**  | object | Defines configuration for HorizontalPodAutoscaler. |
| **components.pilot.k8s.hpaSpec.maxReplicas**  | integer | Specifies the upper limit for the number of Pods that can be set by the autoscaler. It cannot be smaller than **MinReplicas**. |
| **components.pilot.k8s.hpaSpec.minReplicas**  | integer | Specifies the lower limit for the number of replicas to which the autoscaler can scale down. By default, it is set to 1 Pod. The value can be set to 0 if the alpha feature gate HPAScaleToZero is enabled and at least one Object or External metric is configured. Scaling is active as long as at least one metric value is available. |
| **components.pilot.k8s.resources**  | object | Defines [Kubernetes resources configuration](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). For more information, read about Resources in the [Istio documentation](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#Resources). |
| **components.pilot.k8s.strategy**  | object | Defines the rolling update strategy. To learn more, read about DeploymentStrategy in the [Istio documentation](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#DeploymentStrategy). |
| **components.proxy**  | object | Defines component configuration for the Istio proxy sidecar. |
| **components.proxy.k8s.resources**  | object | Defines [Kubernetes resources configuration](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/). To learn more, read about Resources in the [Istio documnetation](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#Resources).|
| **config**  | object | Specifies the configuration for the Istio installation. |
| **config.numTrustedProxies**  | integer | Specifies the number of trusted proxies deployed in front of the Istio gateway proxy. |

**Status:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **state** (required) | string | Signifies the current state of **CustomObject**. Its value can be either `Ready`, `Processing`, `Error`, or `Deleting`. |