package v2

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const fingerprintAnnotationKey = "istio-operator.kyma-project.io/restartHash"

// FingerprintDriftChanged is an Evaluator that detects when a workload's
// fingerprint annotation differs from the current fingerprint value.
type FingerprintDriftChanged struct {
	currentFingerprint string
}

func (e *FingerprintDriftChanged) Name() string {
	return "FingerprintDriftChanged"
}

// Evaluate returns Restart when obj is a workload (Deployment, StatefulSet,
// DaemonSet, or ReplicaSet) and its fingerprint annotation differs from the
// current fingerprint. It returns Continue for non-workload objects, Pods,
// empty current fingerprint, or matching annotations.
func (e *FingerprintDriftChanged) Evaluate(obj Object) Decision {
	if e.currentFingerprint == "" {
		return Continue
	}

	stamp := e.getStampFromWorkload(obj)
	if stamp == nil {
		return Continue
	}

	if *stamp != e.currentFingerprint {
		return Restart
	}
	return Continue
}

// getStampFromWorkload extracts the fingerprint annotation from a workload's
// pod template spec, or nil if the object is not a supported workload type.
// For workloads, it returns a pointer to the annotation value (even if empty).
func (e *FingerprintDriftChanged) getStampFromWorkload(obj Object) *string {
	switch w := obj.(type) {
	case *appsv1.Deployment:
		return e.getAnnotationValue(w.Spec.Template.ObjectMeta.Annotations)
	case *appsv1.StatefulSet:
		return e.getAnnotationValue(w.Spec.Template.ObjectMeta.Annotations)
	case *appsv1.DaemonSet:
		return e.getAnnotationValue(w.Spec.Template.ObjectMeta.Annotations)
	case *appsv1.ReplicaSet:
		return e.getAnnotationValue(w.Spec.Template.ObjectMeta.Annotations)
	case *corev1.Pod:
		return nil
	default:
		return nil
	}
}

// getAnnotationValue retrieves the fingerprint annotation value for a workload.
// It returns a pointer to the annotation value (which may be an empty string if
// the annotation is missing), allowing the caller to detect missing annotations.
func (e *FingerprintDriftChanged) getAnnotationValue(annotations map[string]string) *string {
	stamp := annotations[fingerprintAnnotationKey]
	return &stamp
}

// NewFingerprintDriftChanged returns an Evaluator that detects fingerprint drift
// for workload objects.
func NewFingerprintDriftChanged(currentFingerprint string) Evaluator {
	return &FingerprintDriftChanged{
		currentFingerprint: currentFingerprint,
	}
}
