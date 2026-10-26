package evaluation

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v2 "github.com/kyma-project/istio/operator/pkg/restarter/v2"
)

type (
	// Evaluator aliases the v2 Evaluator so there is a single source of truth.
	Evaluator = v2.Evaluator
	Decision  = v2.Decision
	Object    = v2.Object
)

const (
	Continue = v2.Continue
	Restart  = v2.Restart
	Stop     = v2.Stop
)

var ErrDecisionUnknown = v2.ErrDecisionUnknown

// RestartKind distinguishes which evaluation pass produced a Restart decision.
type RestartKind int

const (
	KindNone        RestartKind = iota // no restart (Continue or Stop)
	KindConfigDrift                    // winning evaluator fired on the workload object
	KindPodState                       // winning evaluator fired on a pod object
)

// EvaluateDetailed runs all evaluators over the workload and then over each of
// its pods, reporting whether the Restart came from a workload-level evaluator
// (KindConfigDrift) or a pod-level evaluator (KindPodState). Returns Stop
// immediately if any evaluator votes Stop, Restart if any votes Restart,
// Continue if all votes are Continue. Returns Continue (KindNone) when pods is
// empty — a workload with no running pods has nothing to restart, so the
// workload-level loop is skipped. Returns KindNone when decision is Continue or Stop.
// The distinction lets callers apply the fingerprint guard only to config-drift
// restarts: a pod-state restart (wrong image after a botched rollout) must not
// be suppressed by a stamp from the previous config-drift restart.
//
// Callers are responsible for only invoking this on workloads that should carry
// an Istio sidecar (see ShouldInject): config-drift evaluators ignore the
// workload object and would otherwise vote Restart on every reconcile against a
// non-mesh workload.
func EvaluateDetailed(ctx context.Context, workload Object, pods []*corev1.Pod, evaluators []Evaluator) (Decision, RestartKind) {
	if len(pods) == 0 {
		return Continue, KindNone
	}
	lg := logf.FromContext(ctx).WithValues("workload", workload.GetName(), "namespace", workload.GetNamespace())

	if d := runEvaluators(ctx, lg, workload, evaluators); d != Continue {
		if d == Restart {
			return Restart, KindConfigDrift
		}
		return d, KindNone
	}
	for _, p := range pods {
		podLg := lg.WithValues("pod", p.GetName())
		podLg.V(2).Info("Evaluating pod")
		if d := runEvaluators(ctx, podLg, p, evaluators); d != Continue {
			if d == Restart {
				return Restart, KindPodState
			}
			return d, KindNone
		}
	}
	lg.V(2).Info("All evaluators voted continue")
	return Continue, KindNone
}

// runEvaluators returns the first non-Continue decision produced by evaluators
// over target. Continue when every evaluator votes Continue.
func runEvaluators(ctx context.Context, lg logr.Logger, target Object, evaluators []Evaluator) Decision {
	for _, e := range evaluators {
		evLg := lg.WithValues("evaluator", e.Name())
		switch e.Evaluate(target) {
		case Stop:
			evLg.V(3).Info("Evaluator voted stop")
			return Stop
		case Restart:
			evLg.V(3).Info("Evaluator voted restart")
			return Restart
		case Continue:
			evLg.V(3).Info("Evaluator voted continue")
		}
	}
	return Continue
}
