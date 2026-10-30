package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
)

const (
	reasonRestarting = "RestartingProxy"
	reasonStopped    = "ProxyRestartStopped"
)

// emitDecisionEvent records a Kubernetes Event on obj for Restart or Stop decisions.
// evaluator is the name of the evaluator that produced the decision.
func emitDecisionEvent(r record.EventRecorder, obj client.Object, d evaluation.Decision, evaluator string) {
	if r == nil {
		return
	}
	switch d {
	case evaluation.Restart:
		r.Eventf(obj, corev1.EventTypeNormal, reasonRestarting, "Istio proxy config or version is outdated (evaluator: %s)", evaluator)
	case evaluation.Stop:
		r.Eventf(obj, corev1.EventTypeWarning, reasonStopped, "Istio proxy evaluation stopped (evaluator: %s)", evaluator)
	}
}
