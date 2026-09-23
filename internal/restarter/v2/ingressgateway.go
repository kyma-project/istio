package v2

import (
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	ingressGatewayNamespace      = "istio-system"
	ingressGatewayDeploymentName = "istio-ingressgateway"
)

// NumTrustedProxiesConfigChanged is an Evaluator that detects a change in the
// number of trusted proxies between the last applied config and the current CR.
type NumTrustedProxiesConfigChanged struct {
	oldNumTrustedProxies *int
	newNumTrustedProxies *int
}

// Evaluate returns Restart when obj is the ingress gateway Deployment and the
// number of trusted proxies changed, and Continue otherwise.
func (c *NumTrustedProxiesConfigChanged) Evaluate(obj Object) Decision {
	return evaluate(obj, c.oldNumTrustedProxies, c.newNumTrustedProxies)
}

// NewNumTrustedProxiesConfigChanged returns an Evaluator that restarts the ingress
// gateway when the number of trusted proxies changed.
func NewNumTrustedProxiesConfigChanged(cr operatorv1alpha2.Istio, lastApplied configuration.AppliedConfig) Evaluator {
	return &NumTrustedProxiesConfigChanged{
		oldNumTrustedProxies: lastApplied.Config.NumTrustedProxies,
		newNumTrustedProxies: cr.Spec.Config.NumTrustedProxies,
	}
}

// ForwardClientCertDetailsConfigChanged is an Evaluator that detects a change in
// the XFCC strategy between the last applied config and the
// current CR.
type ForwardClientCertDetailsConfigChanged struct {
	oldForwardClientCert *operatorv1alpha2.XFCCStrategy
	newForwardClientCert *operatorv1alpha2.XFCCStrategy
}

// Evaluate returns Restart when obj is the ingress gateway Deployment and the
// XFCC strategy changed, and Continue otherwise.
func (c *ForwardClientCertDetailsConfigChanged) Evaluate(obj Object) Decision {
	return evaluate(obj, c.oldForwardClientCert, c.newForwardClientCert)
}

// NewForwardClientCertDetailsConfigChanged returns an Evaluator that restarts the
// ingress gateway when the XFCC strategy changed.
func NewForwardClientCertDetailsConfigChanged(cr operatorv1alpha2.Istio, lastApplied configuration.AppliedConfig) Evaluator {
	return &ForwardClientCertDetailsConfigChanged{
		oldForwardClientCert: lastApplied.Config.ForwardClientCertDetails,
		newForwardClientCert: cr.Spec.Config.ForwardClientCertDetails,
	}
}

// TrustDomainConfigChanged is an Evaluator that detects a change in the mesh trust
// domain between the last applied config and the current CR.
type TrustDomainConfigChanged struct {
	oldTrustDomain *string
	newTrustDomain *string
}

// Evaluate returns Restart when obj is the ingress gateway Deployment and the
// trust domain changed, and Continue otherwise.
func (c *TrustDomainConfigChanged) Evaluate(obj Object) Decision {
	return evaluate(obj, c.oldTrustDomain, c.newTrustDomain)
}

// NewTrustDomainConfigChanged returns an Evaluator that restarts the ingress
// gateway when the trust domain changed.
func NewTrustDomainConfigChanged(cr operatorv1alpha2.Istio, lastApplied configuration.AppliedConfig) Evaluator {
	return &TrustDomainConfigChanged{
		oldTrustDomain: lastApplied.Config.TrustDomain,
		newTrustDomain: cr.Spec.Config.TrustDomain,
	}
}

// evaluate returns Restart when obj is the istio-ingressgateway Deployment and
// the config pointer changed, and Continue otherwise. A nil pointer is treated
// as distinct from any set value (nil equals only nil).
func evaluate[T comparable](obj Object, old, new *T) Decision {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return Continue
	}
	if deployment.GetName() != ingressGatewayDeploymentName ||
		deployment.GetNamespace() != ingressGatewayNamespace {
		return Continue
	}

	var changed bool
	if old == nil || new == nil {
		changed = old != new
	} else {
		changed = *old != *new
	}
	if changed {
		return Restart
	}
	return Continue
}
