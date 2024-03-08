package restarter

import (
	"context"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
)

type Restarter interface {
	Restart(ctx context.Context, istioCR *operatorv1alpha2.Istio) described_errors.DescribedError
}

// Restart invokes the given restarters and returns the most severe error.
func Restart(ctx context.Context, istioCR *operatorv1alpha2.Istio, restarters []Restarter) described_errors.DescribedError {
	var restarterErrs []described_errors.DescribedError

	for _, r := range restarters {
		err := r.Restart(ctx, istioCR)
		if err != nil {
			restarterErrs = append(restarterErrs, err)
		}
	}

	return described_errors.GetMostSevereErr(restarterErrs)
}
