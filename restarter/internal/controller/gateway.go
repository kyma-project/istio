package controller

import (
	"context"
	"fmt"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	"github.com/kyma-project/istio/operator/restarter/internal/restart"
)

const (
	gatewayLabelKey          = "istio"
	ingressGatewayLabelValue = "ingressgateway"
	egressGatewayLabelValue  = "egressgateway"
)

func isIstioGateway(d *appsv1.Deployment) bool {
	if d == nil {
		return false
	}

	return slices.Contains(
		[]string{ingressGatewayLabelValue, egressGatewayLabelValue},
		d.Labels[gatewayLabelKey],
	)
}

// reconcileGateway evaluates an Istio gateway Deployment (ingress or egress)
// and restarts it when its gateway-config fingerprint has drifted. It mirrors the
// reconcileWorkload path but operates on a concrete *appsv1.Deployment, calls
// GatewaySnapshot instead of Snapshot, and has no ShouldInject gate.
func reconcileGateway(
	ctx context.Context,
	c client.Client,
	evaluators evaluation.Source,
	recorder record.EventRecorder,
	d *appsv1.Deployment,
) (ctrl.Result, error) {
	l := logf.FromContext(ctx).WithValues("deployment", d.Name, "namespace", d.Namespace)
	l.V(2).Info("Reconciling gateway")

	var pods []*corev1.Pod
	if d.Spec.Selector != nil &&
		(len(d.Spec.Selector.MatchLabels) > 0 || len(d.Spec.Selector.MatchExpressions) > 0) {
		sel, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("invalid gateway deployment selector: %w", err)
		}
		podList := &corev1.PodList{}
		if err := c.List(ctx, podList,
			client.InNamespace(d.Namespace),
			client.MatchingLabelsSelector{Selector: sel},
		); err != nil {
			l.Error(err, "Failed to list gateway pods")
			return ctrl.Result{}, err
		}
		pods = make([]*corev1.Pod, len(podList.Items))
		for i := range podList.Items {
			pods[i] = &podList.Items[i]
		}
	}

	gatewayEvs, gatewayFingerprint, err := evaluators.GatewaySnapshot(ctx)
	if err != nil {
		l.Error(err, "Failed to snapshot gateway evaluators")
		return ctrl.Result{}, err
	}

	if gatewayFingerprint == "" {
		l.V(2).Info("Gateway fingerprint empty; skipping")
		return ctrl.Result{}, nil
	}

	decision, kind, evaluator := evaluation.EvaluateDetailed(ctx, d, pods, gatewayEvs)
	l.V(2).Info("Gateway evaluation decision", "decision", decision, "evaluator", evaluator, "pods", len(pods))
	emitDecisionEvent(recorder, d, decision, evaluator)

	if decision == evaluation.Restart {
		if kind == evaluation.KindConfigDrift && restart.AlreadyRestartedForConfig(d.Spec.Template.Annotations[restart.HashAnnotation], gatewayFingerprint) {
			l.V(1).Info("Gateway already restarted for current config; skipping", "fingerprint", gatewayFingerprint)
			return ctrl.Result{}, nil
		}
		if !restart.DeploymentRolledOut(d) {
			l.V(1).Info("Gateway rollout in progress; deferring restart")
			return ctrl.Result{}, nil
		}
		l.V(1).Info("Restarting gateway", "fingerprint", gatewayFingerprint)
		if err := restart.Deployment(ctx, c, d, gatewayFingerprint); err != nil {
			l.Error(err, "Failed to restart gateway")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
