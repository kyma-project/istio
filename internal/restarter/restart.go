package restarter

import (
	"context"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
)

// Restarter is an interface for restarting Istio components.
// It uses predicates to evaluate if the restart is needed.
// If the evaluation returns true, the restarter restarts the component.
// Additional boolean return parameter indicates if the reconciliation should be requeued.
type Restarter interface {
	Restart(ctx context.Context, istioCR *operatorv1alpha2.Istio) (describederrors.DescribedError, bool)
}

// Restart invokes the given restarters and returns the most severe error.
func Restart(ctx context.Context, istioCR *operatorv1alpha2.Istio, restarters []Restarter) (describederrors.DescribedError, bool) {
	var restarterErrs []describederrors.DescribedError

	needsRequeue := false
	for _, r := range restarters {
		err, requeue := r.Restart(ctx, istioCR)
		needsRequeue = requeue || needsRequeue
		if err != nil {
			restarterErrs = append(restarterErrs, err)
		}
	}

	return describederrors.GetMostSevereErr(restarterErrs), needsRequeue
}
