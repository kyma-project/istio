package ingressgateway

import (
	"context"
	"encoding/json"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/pkg/labels"
)

type RestartPredicate struct {
	istioCR *operatorv1alpha2.Istio
}

func NewRestartPredicate(istioCR *operatorv1alpha2.Istio) *RestartPredicate {
	return &RestartPredicate{istioCR: istioCR}
}

type appliedConfig struct {
	operatorv1alpha2.IstioSpec
	IstioTag string
}

func getLastAppliedConfiguration(istioCR *operatorv1alpha2.Istio) (appliedConfig, error) {
	lastAppliedConfig := appliedConfig{}
	if len(istioCR.Annotations) == 0 {
		return lastAppliedConfig, nil
	}

	if lastAppliedAnnotation, found := istioCR.Annotations[labels.LastAppliedConfiguration]; found {
		err := json.Unmarshal([]byte(lastAppliedAnnotation), &lastAppliedConfig)
		if err != nil {
			return lastAppliedConfig, err
		}
	}

	return lastAppliedConfig, nil
}

func (i RestartPredicate) NewIngressGatewayEvaluator(_ context.Context) (filter.IngressGatewayRestartEvaluator, error) {
	lastAppliedConfig, err := getLastAppliedConfiguration(i.istioCR)
	if err != nil {
		return nil, err
	}

	return NumTrustedProxiesRestartEvaluator{
		NewNumTrustedProxies: i.istioCR.Spec.Config.NumTrustedProxies,
		OldNumTrustedProxies: lastAppliedConfig.IstioSpec.Config.NumTrustedProxies,
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
