package v1alpha1

// Type definitions are based on k8s documention and Istio's reference from:
// https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/

type GatewayTopology struct {
	// Defines the number of trusted proxies deployed in front of the Istio gateway proxy.
	// +kubebuilder:validation:Optional
	NumTrustedProxies int `json:"numTrustedProxies,omitempty"`
}

type MeshConfig struct {
	// Mesh configurion.
	// +kubebuilder:validation:Optional
	GatewayTopology GatewayTopology `json:"gatewayTopology,omitempty"`
}

type ResourceMetricSource struct {
	// name is the name of the resource in question.
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// targetAverageUtilization is the target value of the average of the
	// resource metric across all relevant pods, represented as a percentage of
	// the requested value of the resource for the pods.
	// +kubebuilder:validation:Optional
	TargetAverageUtilization int `json:"targetAverageUtilization,omitempty"`
}

type MetricSpec struct {
	// type is the type of metric source.  It should be one of "ContainerResource", "External",
	// "Object", "Pods" or "Resource", each mapping to a matching field in the object.
	// +kubebuilder:validation:Optional
	Type string `json:"type,omitempty"`

	// resource refers to a resource metric (such as those specified in
	// requests and limits) known to Kubernetes describing each pod in the
	// current scale target (e.g. CPU or memory). Such metrics are built in to
	// Kubernetes, and have special scaling options on top of those available
	// to normal per-pod metrics using the "pods" source.
	// +kubebuilder:validation:Optional
	Resource ResourceMetricSource `json:"resource,omitempty"`
}

type HorizontalPodAutoscalerSpec struct {
	// minReplicas is the lower limit for the number of replicas to which the autoscaler
	// can scale down. It defaults to 1 pod. minReplicas is allowed to be 0 if the
	// alpha feature gate HPAScaleToZero is enabled and at least one Object or External
	// metric is configured. Scaling is active as long as at least one metric value is
	// available.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MinReplicas int `json:"minReplicas,omitempty"`

	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.
	// It cannot be less that minReplicas.
	// +kubebuilder:validation:Optional
	MaxReplicas int `json:"maxReplicas,omitempty"`

	// metrics contains the specifications for which to use to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used). The desired replica count is calculated multiplying the
	// ratio between the target value and the current value by the current
	// number of pods. Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa. See the individual metric source types for
	// more information about how each type of metric must respond.
	// If not set, the default metric will be set to 80% average CPU utilization.
	// +kubebuilder:validation:Optional
	Metrics []MetricSpec `json:"metrics,omitempty"`
}

type RollingUpdateDeployment struct {
	// The maximum number of pods that can be scheduled above the desired number of
	// pods.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// This can not be 0 if MaxUnavailable is 0.
	// Absolute number is calculated from percentage by rounding up.
	// By default, a value of 1 is used.
	// Example: when this is set to 30%, the new RC can be scaled up immediately when
	// the rolling update starts, such that the total number of old and new pods do not exceed
	// 130% of desired pods. Once old pods have been killed,
	// new RC can be scaled up further, ensuring that total number of pods running
	// at any time during the update is at most 130% of desired pods.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9][\d]?%|100%|\d+$
	MaxSurge string `json:"maxSurge,omitempty"`

	// The maximum number of pods that can be unavailable during the update.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// Absolute number is calculated from percentage by rounding down.
	// This can not be 0 if MaxSurge is 0.
	// By default, a fixed value of 1 is used.
	// Example: when this is set to 30%, the old RC can be scaled down to 70% of desired pods
	// immediately when the rolling update starts. Once new pods are ready, old RC
	// can be scaled down further, followed by scaling up the new RC, ensuring
	// that the total number of pods available at all times during the update is at
	// least 70% of desired pods.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9][\d]?%|100%|\d+$
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
}

type DeploymentStrategy struct {
	// Spec to control the desired behavior of rolling update.
	// +kubebuilder:validation:Optional
	RollingUpdate RollingUpdateDeployment `json:"rollingUpdate,omitempty"`
}

type ResourceSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9](\d?)+(m|g)$
	CPU string `json:"cpu,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9](\d?)+(Mi|Gi)$
	Memory string `json:"memory,omitempty"`
}

type Resources struct {
	// Limits describes the maximum amount of compute resources allowed.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +kubebuilder:validation:Optional
	Limits ResourceSpec `json:"limits,omitempty"`

	// Requests describes the minimum amount of compute resources required.
	// If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
	// otherwise to an implementation-defined value.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +kubebuilder:validation:Optional
	Requests ResourceSpec `json:"requests,omitempty"`
}

type Deployment struct {
	// hpaSpec describes the desired functionality of the HorizontalPodAutoscaler.
	// +kubebuilder:validation:Optional
	HpaSpec HorizontalPodAutoscalerSpec `json:"hpa,omitempty"`

	// The deployment strategy to use to replace existing pods with new ones.
	// +kubebuilder:validation:Optional
	Strategy DeploymentStrategy `json:"strategy,omitempty"`

	// Resources describes the compute resource requirements.
	// +kubebuilder:validation:Optional
	Resources Resources `json:"resources,omitempty"`
}

type Istiod struct {
	// Deployment enables declarative updates for Pods and ReplicaSets.
	// +kubebuilder:validation:Optional
	Deployment Deployment `json:"deployment,omitempty"`
}

type IngressGateway struct {
	// Deployment enables declarative updates for Pods and ReplicaSets.
	// +kubebuilder:validation:Optional
	Deployment Deployment `json:"deployment,omitempty"`
}

type Controlplane struct {
	// +kubebuilder:validation:Optional
	MeshConfig MeshConfig `json:"meshConfig,omitempty"`

	// +kubebuilder:validation:Optional
	Istiod Istiod `json:"istiod,omitempty"`
}

type Dataplane struct {
	// +kubebuilder:validation:Optional
	IngressGateway IngressGateway `json:"ingressGateway,omitempty"`
}
