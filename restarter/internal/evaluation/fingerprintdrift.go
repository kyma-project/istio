package evaluation

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// FingerprintAnnotationKey is the pod-template annotation key holding the
// restart fingerprint hash. It is the single source of truth for that key.
const FingerprintAnnotationKey = "istio-operator.kyma-project.io/restartHash"

// FingerprintChanged is an Evaluator that detects when a workload's
// fingerprint annotation differs from the current fingerprint value.
type FingerprintChanged struct {
	currentFingerprint string
}

func (e *FingerprintChanged) Name() string {
	return "FingerprintChanged"
}

// Evaluate returns Restart when obj is a workload (Deployment, StatefulSet,
// DaemonSet, or ReplicaSet) and its fingerprint annotation differs from the
// current fingerprint. It returns Continue for non-workload objects, Pods,
// or matching annotations.
func (e *FingerprintChanged) Evaluate(obj Object) Decision {
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
func (e *FingerprintChanged) getStampFromWorkload(obj Object) *string {
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
func (e *FingerprintChanged) getAnnotationValue(annotations map[string]string) *string {
	return new(annotations[FingerprintAnnotationKey])
}

// NewFingerprintChanged returns an Evaluator that detects fingerprint drift
// for workload objects.
func NewFingerprintChanged(currentFingerprint string) Evaluator {
	return &FingerprintChanged{
		currentFingerprint: currentFingerprint,
	}
}
