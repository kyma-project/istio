# Enable Control Plane VPA

Enable Control Plane VPA to allow automatic memory scaling of Istio control plane components in large workload deployments, avoiding manual tuning of memory requests and limits.

> [!NOTE]
> **Prerequisite:** The cluster must have the VPA CustomResourceDefinition (CRD) `verticalpodautoscalers.autoscaling.k8s.io` installed. If the CRD is not present, the VPA resources are silently skipped.

When you set `enableControlPlaneVPA` to `true` in the `istio-features` ConfigMap, the Istio module creates [VerticalPodAutoscaler (VPA)](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) resources for the following Istio control plane components: istiod, ingress gateway, egress gateway, and CNI DaemonSet.

The VPA manages memory resources only, allowing it to coexist safely with the existing HorizontalPodAutoscaler (HPA) that scales based on CPU utilization. This enables automatic memory optimization for large-scale mesh deployments.

Enabling this field has the following effects:

- VPA resources are created in the `istio-system` namespace targeting istiod, istio-ingressgateway, istio-egressgateway, and istio-cni-node.
- Each VPA uses `updateMode: InPlaceOrRecreate` for non-disruptive scaling where supported.
- Only memory requests and limits are managed (`controlledResources: [memory]`).
- Any memory-based metrics in the HPA are automatically removed to prevent autoscaler conflicts.
