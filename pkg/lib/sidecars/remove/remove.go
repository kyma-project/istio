package remove

import (
	"context"

	"github.com/go-logr/logr"
	podInfo "github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SidecarRemover struct {
	ctx       context.Context
	k8sclient client.Client
	logger    *logr.Logger
}

func NewSidecarRemover(ctx context.Context, k8sclient client.Client, logger *logr.Logger) SidecarRemover {
	return SidecarRemover{
		ctx:       ctx,
		k8sclient: k8sclient,
		logger:    logger,
	}
}

func (r *SidecarRemover) RemoveSidecars() ([]restart.RestartWarning, error) {
	toRestart, err := podInfo.GetAllInjectedPods(r.ctx, r.k8sclient)
	if err != nil {
		return nil, err
	}

	return restart.Restart(r.ctx, r.k8sclient, *toRestart, r.logger)
}
