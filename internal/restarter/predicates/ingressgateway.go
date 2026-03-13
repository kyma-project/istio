package predicates

import (
	"context"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

type RestartPredicate struct {
	istioCR *operatorv1alpha2.Istio
}

func NewIngressGatewayRestartPredicate(istioCR *operatorv1alpha2.Istio) *RestartPredicate {
	return &RestartPredicate{istioCR: istioCR}
}

func (i RestartPredicate) NewIngressGatewayEvaluator(_ context.Context) (IngressGatewayRestartEvaluator, error) {
	lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(i.istioCR)
	if err != nil {
		return nil, err
	}

	return CompositeIngressGatewayRestartEvaluator{
		Evaluators: []IngressGatewayRestartEvaluator{
			NumTrustedProxiesRestartEvaluator{
				NewNumTrustedProxies: i.istioCR.Spec.Config.NumTrustedProxies,
				OldNumTrustedProxies: lastAppliedConfig.Config.NumTrustedProxies,
			},
			XForwardClientCertRestartEvaluator{
				NewXForwardClientCert: i.istioCR.Spec.Config.ForwardClientCertDetails,
				OldXForwardClientCert: lastAppliedConfig.Config.ForwardClientCertDetails,
			},
			TrustDomainsRestartEvaluator{
				NewTrustDomain: i.istioCR.Spec.Config.TrustDomain,
				OldTrustDomain: lastAppliedConfig.Config.TrustDomain,
			},
		},
	}, nil
}

type CompositeIngressGatewayRestartEvaluator struct {
	Evaluators []IngressGatewayRestartEvaluator
}

func (c CompositeIngressGatewayRestartEvaluator) RequiresIngressGatewayRestart() bool {
	for _, evaluator := range c.Evaluators {
		if evaluator.RequiresIngressGatewayRestart() {
			return true
		}
	}

	return false
}

type NumTrustedProxiesRestartEvaluator struct {
	NewNumTrustedProxies *int
	OldNumTrustedProxies *int
}

type XForwardClientCertRestartEvaluator struct {
	NewXForwardClientCert *operatorv1alpha2.XFCCStrategy
	OldXForwardClientCert *operatorv1alpha2.XFCCStrategy
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

type TrustDomainsRestartEvaluator struct {
	NewTrustDomain *string
	OldTrustDomain *string
}

func (i TrustDomainsRestartEvaluator) RequiresIngressGatewayRestart() bool {
	isNewNotNil := i.NewTrustDomain != nil
	isOldNotNil := i.OldTrustDomain != nil
	if isNewNotNil && isOldNotNil && *i.NewTrustDomain != *i.OldTrustDomain {
		return true
	}
	if isNewNotNil != isOldNotNil {
		return true
	}
	return false
}

func (i XForwardClientCertRestartEvaluator) RequiresIngressGatewayRestart() bool {
	isNewNotNil := i.NewXForwardClientCert != nil
	isOldNotNil := i.OldXForwardClientCert != nil
	if isNewNotNil && isOldNotNil && *i.NewXForwardClientCert != *i.OldXForwardClientCert {
		return true
	} else if isNewNotNil != isOldNotNil {
		return true
	}

	return false
}
