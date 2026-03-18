package istioresources

import (
	"context"
	_ "embed"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kyma-project/istio/operator/internal/resources"
)

//go:embed vpa.yaml
var vpaManifest []byte

const vpaCRDName = "verticalpodautoscalers.autoscaling.k8s.io"

type VPA struct {
	shouldDelete bool
}

func NewVPA(shouldDelete bool) VPA {
	return VPA{shouldDelete: shouldDelete}
}

func (v VPA) reconcile(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	vpaAvailable, err := isVPACRDPresent(ctx, k8sClient)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if !vpaAvailable {
		return controllerutil.OperationResultNone, nil
	}

	if v.shouldDelete {
		return resources.DeleteIfPresent(ctx, k8sClient, vpaManifest)
	}

	return resources.Apply(ctx, k8sClient, vpaManifest, &owner)
}

func (VPA) Name() string {
	return "VerticalPodAutoscaler/istio-operator-vpa"
}

func isVPACRDPresent(ctx context.Context, k8sClient client.Client) (bool, error) {
	var crd unstructured.Unstructured
	crd.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})

	err := k8sClient.Get(ctx, types.NamespacedName{Name: vpaCRDName}, &crd)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
