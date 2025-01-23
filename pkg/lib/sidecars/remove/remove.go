package remove

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RemoveSidecars(ctx context.Context, k8sclient client.Client, logger *logr.Logger) ([]restart.RestartWarning, error) {
	podsLister := pods.NewPods(k8sclient, logger)
	toRestart, err := podsLister.GetAllInjectedPods(ctx)
	if err != nil {
		return nil, err
	}

	return restart.Restart(ctx, k8sclient, toRestart, logger, false)
}
