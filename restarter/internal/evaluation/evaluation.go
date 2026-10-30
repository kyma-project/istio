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

// RestartKind distinguishes which evaluation pass produced a Restart decision.
type RestartKind int

const (
	KindNone RestartKind = iota
	KindConfigDrift
	KindPodState
)

// EvaluateDetailed returns the decision, which evaluation pass produced it, and
// the name of the evaluator that produced a non-Continue decision (empty when the
// decision is Continue).
func EvaluateDetailed(ctx context.Context, workload Object, pods []*corev1.Pod, evaluators []Evaluator) (Decision, RestartKind, string) {
	if len(pods) == 0 {
		return Continue, KindNone, ""
	}
	lg := logf.FromContext(ctx).WithValues("workload", workload.GetName(), "namespace", workload.GetNamespace())

	if d, name := runEvaluators(lg, workload, evaluators); d != Continue {
		if d == Restart {
			return Restart, KindConfigDrift, name
		}
		return d, KindNone, name
	}
	for _, p := range pods {
		podLg := lg.WithValues("pod", p.GetName())
		podLg.V(2).Info("Evaluating pod")
		if d, name := runEvaluators(podLg, p, evaluators); d != Continue {
			if d == Restart {
				return Restart, KindPodState, name
			}
			return d, KindNone, name
		}
	}
	lg.V(2).Info("All evaluators voted continue")
	return Continue, KindNone, ""
}

// runEvaluators returns the first non-Continue decision produced by evaluators
// over target along with the deciding evaluator's name. Continue with an empty
// name when every evaluator votes Continue.
func runEvaluators(lg logr.Logger, target Object, evaluators []Evaluator) (Decision, string) {
	for _, e := range evaluators {
		evLg := lg.WithValues("evaluator", e.Name())
		switch e.Evaluate(target) {
		case Stop:
			evLg.V(3).Info("Evaluator voted stop")
			return Stop, e.Name()
		case Restart:
			evLg.V(1).Info("Evaluator voted restart")
			return Restart, e.Name()
		case Continue:
			evLg.V(3).Info("Evaluator voted continue")
		}
	}
	return Continue, ""
}
