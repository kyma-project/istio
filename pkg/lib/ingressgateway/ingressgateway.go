package ingressgateway

import (
	"context"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
)

type RestartPredicate struct {
	istioCR *operatorv1alpha2.Istio
}

func NewRestartPredicate(istioCR *operatorv1alpha2.Istio) *RestartPredicate {
	return &RestartPredicate{istioCR: istioCR}
}

func (i RestartPredicate) NewIngressGatewayEvaluator(_ context.Context) (filter.IngressGatewayRestartEvaluator, error) {
	lastAppliedConfig, err := istio.GetLastAppliedConfiguration(i.istioCR)
	if err != nil {
		return nil, err
	}

	return NumTrustedProxiesRestartEvaluator{
		NewNumTrustedProxies: i.istioCR.Spec.Config.NumTrustedProxies,
		OldNumTrustedProxies: lastAppliedConfig.Config.NumTrustedProxies,
	}, nil
}

type NumTrustedProxiesRestartEvaluator struct {
	NewNumTrustedProxies *int
	OldNumTrustedProxies *int
}

func (i NumTrustedProxiesRestartEvaluator) RequiresIngressGatewayRestart() bool {
	isNewNotNil := i.NewNumTrustedProxies != nil
	isOldNotNil := i.OldNumTrustedProxies != nil
	if isNewNotNil && isOldNotNil && *i.NewNumTrustedProxies != *i.OldNumTrustedProxies {
		return true
	} else if isNewNotNil != isOldNotNil {
		return true
	}

	return false
}
