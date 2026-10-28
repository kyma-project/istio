package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/restarter/internal/evaluation"
)

const (
	reasonRestartRequired = "RestartRequired"
	reasonRestartSkipped  = "RestartSkipped"
)

// emitDecisionEvent records a Kubernetes Event on obj for Restart or Stop decisions.
func emitDecisionEvent(r record.EventRecorder, obj client.Object, d evaluation.Decision) {
	if r == nil {
		return
	}
	switch d {
	case evaluation.Restart:
		r.Event(obj, corev1.EventTypeWarning, reasonRestartRequired, "workload pods require a restart")
	case evaluation.Stop:
		r.Event(obj, corev1.EventTypeNormal, reasonRestartSkipped, "restart skipped: evaluation returned Stop")
	}
}
