package istioresources

import (
	"context"
	_ "embed"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kyma-project/istio/operator/internal/resources"
)

//go:embed control_plane_vpas/istiod.yaml
var controlPlaneVPAIstiod []byte

//go:embed control_plane_vpas/ingress-gateway.yaml
var controlPlaneVPAIngressGateway []byte

//go:embed control_plane_vpas/egress-gateway.yaml
var controlPlaneVPAEgressGateway []byte

//go:embed control_plane_vpas/cni.yaml
var controlPlaneVPACni []byte

type ControlPlaneVPA struct {
	shouldDelete bool
}

func NewControlPlaneVPA(shouldDelete bool) ControlPlaneVPA {
	return ControlPlaneVPA{shouldDelete: shouldDelete}
}

func (v ControlPlaneVPA) reconcile(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	vpaAvailable, err := isVPACRDPresent(ctx, k8sClient)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if !vpaAvailable {
		return controllerutil.OperationResultNone, nil
	}

	manifests := [][]byte{
		controlPlaneVPAIstiod,
		controlPlaneVPAIngressGateway,
		controlPlaneVPAEgressGateway,
		controlPlaneVPACni,
	}

	if v.shouldDelete {
		return deleteAll(ctx, k8sClient, manifests)
	}

	return applyAll(ctx, k8sClient, manifests, nil)
}

func (ControlPlaneVPA) Name() string {
	return "VerticalPodAutoscaler/control-plane"
}

func applyAll(ctx context.Context, k8sClient client.Client, manifests [][]byte, owner *metav1.OwnerReference) (controllerutil.OperationResult, error) {
	combinedResult := controllerutil.OperationResultNone
	for _, manifest := range manifests {
		result, err := resources.Apply(ctx, k8sClient, manifest, owner)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
		if result != controllerutil.OperationResultNone {
			combinedResult = result
		}
	}
	return combinedResult, nil
}

func deleteAll(ctx context.Context, k8sClient client.Client, manifests [][]byte) (controllerutil.OperationResult, error) {
	combinedResult := controllerutil.OperationResultNone
	for _, manifest := range manifests {
		result, err := resources.DeleteIfPresent(ctx, k8sClient, manifest)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
		if result != controllerutil.OperationResultNone {
			combinedResult = result
		}
	}
	return combinedResult, nil
}
