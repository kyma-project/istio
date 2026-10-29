package v2

import (
	"slices"

	appsv1 "k8s.io/api/apps/v1"
)

const gatewayLabelKey = "istio"

var gatewayLabelValues = []string{"ingressgateway", "egressgateway"}

// GatewayFingerprintDriftChanged is an Evaluator that detects when an Istio
// gateway Deployment's fingerprint annotation differs from the current
// fingerprint value. It acts on Deployments labelled istio=ingressgateway or
// istio=egressgateway to provide defense-in-depth isolation.
type GatewayFingerprintDriftChanged struct {
	currentFingerprint string
}

func (e *GatewayFingerprintDriftChanged) Name() string {
	return "GatewayFingerprintDriftChanged"
}

// Evaluate returns Restart when obj is an Istio gateway Deployment (labelled
// istio=ingressgateway or istio=egressgateway) and its fingerprint annotation
// differs from the current fingerprint. It returns Continue for empty current
// fingerprint, non-Deployment objects, non-gateway Deployments, Pods, or
// matching annotations.
func (e *GatewayFingerprintDriftChanged) Evaluate(obj Object) Decision {
	if e.currentFingerprint == "" {
		return Continue
	}

	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return Continue
	}

	if !slices.Contains(gatewayLabelValues, deployment.GetLabels()[gatewayLabelKey]) {
		return Continue
	}

	stamp := e.getStamp(deployment)

	if *stamp != e.currentFingerprint {
		return Restart
	}
	return Continue
}

// getStamp extracts the fingerprint annotation from the Deployment's pod
// template. Returns a pointer to the value; the empty string when absent (nil
// map or missing key) so that an unstamped gateway triggers a restart.
func (e *GatewayFingerprintDriftChanged) getStamp(deployment *appsv1.Deployment) *string {
	return new(deployment.Spec.Template.ObjectMeta.Annotations[fingerprintAnnotationKey])
}

// NewGatewayFingerprintDriftChanged returns an Evaluator that detects fingerprint
// drift for Istio gateway Deployments.
func NewGatewayFingerprintDriftChanged(currentFingerprint string) Evaluator {
	return &GatewayFingerprintDriftChanged{
		currentFingerprint: currentFingerprint,
	}
}
