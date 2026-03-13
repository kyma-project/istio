package istioresources

import (
	"context"
	_ "embed"

	"github.com/kyma-project/istio/operator/internal/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed istio_edit_clusterrole.yaml
var editClusterRole []byte

//go:embed istio_view_clusterrole.yaml
var viewClusterRole []byte

type ClusterRoles struct {
	shouldDelete bool
}

func (r ClusterRoles) Name() string {
	return "ClusterRoles"
}

func (r ClusterRoles) reconcile(ctx context.Context, c client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	rawRoles := [][]byte{
		editClusterRole,
		viewClusterRole,
	}
	aggrOp := controllerutil.OperationResultNone
	for _, rawRole := range rawRoles {
		op, err := r.reconcileSingleRole(ctx, c, owner, rawRole)
		if err != nil {
			return op, err
		}
		// catch the first non-None operation result, if any
		if aggrOp == controllerutil.OperationResultNone {
			aggrOp = op
		}
	}
	return aggrOp, nil
}

func NewClusterRolesReconciler(shouldDelete bool) ClusterRoles {
	return ClusterRoles{
		shouldDelete: shouldDelete,
	}
}

func (r ClusterRoles) reconcileSingleRole(ctx context.Context, c client.Client, owner metav1.OwnerReference, rawObj []byte) (controllerutil.OperationResult, error) {
	if r.shouldDelete {
		return resources.DeleteIfPresent(ctx, c, rawObj)
	}
	return resources.Apply(ctx, c, rawObj, &owner)
}
