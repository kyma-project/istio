package ingressgateway

import (
	"context"
	"encoding/json"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/internal/filter"
)

type IngressGatewayRestartPredicate struct {
	istioCR *operatorv1alpha2.Istio
}

func NewIngressGatewayRestartPredicate(istioCR *operatorv1alpha2.Istio) *IngressGatewayRestartPredicate {
	return &IngressGatewayRestartPredicate{istioCR: istioCR}
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

	if lastAppliedAnnotation, found := istioCR.Annotations[v1.LastAppliedConfigAnnotation]; found {
		err := json.Unmarshal([]byte(lastAppliedAnnotation), &lastAppliedConfig)
		if err != nil {
			return lastAppliedConfig, err
		}
	}

	return lastAppliedConfig, nil
}

func (i IngressGatewayRestartPredicate) NewIngressGatewayEvaluator(ctx context.Context) (filter.IngressGatewayRestartEvaluator, error) {
	lastAppliedConfig, err := getLastAppliedConfiguration(i.istioCR)
	if err != nil {
		return nil, err
	}

	return IngressGatewayRestartEvaluator{
		newNumTrustedProxies: i.istioCR.Spec.Config.NumTrustedProxies,
		oldNumTrustedProxies: lastAppliedConfig.IstioSpec.Config.NumTrustedProxies,
	}, nil
}

type IngressGatewayRestartEvaluator struct {
	newNumTrustedProxies *int
	oldNumTrustedProxies *int
}

func (i IngressGatewayRestartEvaluator) RequiresIngressGatewayRestart() bool {
	isNewNotNil := i.newNumTrustedProxies != nil
	isOldNotNil := i.oldNumTrustedProxies != nil
	if isNewNotNil && isOldNotNil && *i.newNumTrustedProxies != *i.oldNumTrustedProxies {
		return true
	} else if isNewNotNil != isOldNotNil {
		return true
	}

	return false
}
