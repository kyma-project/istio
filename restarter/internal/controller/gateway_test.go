package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	"github.com/kyma-project/istio/operator/restarter/internal/restart"
)

const (
	defaultIstioGatewayName      = "istio-ingressgateway"
	defaultIstioGatewayNamespace = "istio-system"
)

// ingressGatewayDeployment returns a fully-populated istio-ingressgateway
// Deployment with the given pod-template annotations. rolledOut controls whether
// the Deployment's status satisfies DeploymentRolledOut.
func ingressGatewayDeployment(tmplAnnotations map[string]string, rolledOut bool) *appsv1.Deployment {
	const replicas = int32(1)
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultIstioGatewayName,
			Namespace: defaultIstioGatewayNamespace,
			Labels:    map[string]string{gatewayLabelKey: ingressGatewayLabelValue},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 { ; return new(replicas) }(),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{gatewayLabelKey: ingressGatewayLabelValue},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{gatewayLabelKey: ingressGatewayLabelValue},
					Annotations: tmplAnnotations,
				},
			},
		},
	}
	if rolledOut {
		d.Status = appsv1.DeploymentStatus{
			Replicas:          replicas,
			UpdatedReplicas:   replicas,
			AvailableReplicas: replicas,
		}
	} else {
		// Generation ahead of ObservedGeneration → DeploymentRolledOut returns false.
		d.Generation = 2
		d.Status = appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			Replicas:           replicas,
			UpdatedReplicas:    replicas,
			AvailableReplicas:  replicas,
		}
	}
	return d
}

// gatewayPod returns a pod whose labels match the ingressgateway selector.
func gatewayPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: defaultIstioGatewayNamespace,
			Labels:    map[string]string{gatewayLabelKey: ingressGatewayLabelValue},
		},
	}
}

func TestReconcileGateway(t *testing.T) {
	const fp = "gw-config-fingerprint-abc"

	tests := []struct {
		name           string
		deployment     *appsv1.Deployment
		pods           []client.Object
		gatewayEvs     []evaluation.Evaluator
		gatewayFP      string
		wantRestarted  bool
		wantErrPresent bool
	}{
		{
			name:          "FP differs from template stamp, pods present, rolled out → restart",
			deployment:    ingressGatewayDeployment(nil, true),
			pods:          []client.Object{gatewayPod("gw-pod-1")},
			gatewayEvs:    []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:     fp,
			wantRestarted: true,
		},
		{
			name: "template stamp already equals FP → no restart",
			deployment: ingressGatewayDeployment(map[string]string{
				restart.HashAnnotation:    fp,
				restart.RestartAnnotation: "2020-01-01T00:00:00Z",
			}, true),
			pods:          []client.Object{gatewayPod("gw-pod-1")},
			gatewayEvs:    []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:     fp,
			wantRestarted: false,
		},
		{
			name:          "zero pods → no restart",
			deployment:    ingressGatewayDeployment(nil, true),
			pods:          nil,
			gatewayEvs:    []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:     fp,
			wantRestarted: false,
		},
		{
			name:          "empty gateway fingerprint → no restart",
			deployment:    ingressGatewayDeployment(nil, true),
			pods:          []client.Object{gatewayPod("gw-pod-1")},
			gatewayEvs:    []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:     "",
			wantRestarted: false,
		},
		{
			name:          "rollout in progress → no restart this pass",
			deployment:    ingressGatewayDeployment(nil, false),
			pods:          []client.Object{gatewayPod("gw-pod-1")},
			gatewayEvs:    []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:     fp,
			wantRestarted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			_ = corev1.AddToScheme(s)

			objs := []client.Object{tt.deployment}
			objs = append(objs, tt.pods...)

			c := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(objs...).
				Build()

			src := fakeSource{
				gatewayEvs: tt.gatewayEvs,
				gatewayFP:  tt.gatewayFP,
			}
			rec := record.NewFakeRecorder(10)

			_, err := reconcileGateway(context.Background(), c, src, rec, tt.deployment)
			if (err != nil) != tt.wantErrPresent {
				t.Fatalf("reconcileGateway() error = %v, wantErrPresent %v", err, tt.wantErrPresent)
			}

			got := &appsv1.Deployment{}
			if err := c.Get(context.Background(), types.NamespacedName{
				Name:      defaultIstioGatewayName,
				Namespace: defaultIstioGatewayNamespace,
			}, got); err != nil {
				t.Fatalf("get deployment: %v", err)
			}

			if tt.wantRestarted {
				if got.Spec.Template.Annotations[restart.HashAnnotation] != tt.gatewayFP {
					t.Errorf("HashAnnotation = %q, want %q", got.Spec.Template.Annotations[restart.HashAnnotation], tt.gatewayFP)
				}
				assertRestartAnnotationRecent(t, got.Spec.Template.Annotations[restart.RestartAnnotation])
			} else {
				if got.Spec.Template.Annotations[restart.RestartAnnotation] != "" &&
					got.Spec.Template.Annotations[restart.RestartAnnotation] != "2020-01-01T00:00:00Z" {
					t.Errorf("unexpected restart: restartedAt = %q", got.Spec.Template.Annotations[restart.RestartAnnotation])
				}
			}
		})
	}
}

func TestIsIstioGateway(t *testing.T) {
	tests := []struct {
		name     string
		deploy   *appsv1.Deployment
		expected bool
	}{
		{
			name:     "nil deployment",
			deploy:   nil,
			expected: false,
		},
		{
			name: "ingressgateway label on deployment",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-ingressgateway",
					Namespace: "istio-system",
					Labels: map[string]string{
						"istio": "ingressgateway",
					},
				},
			},
			expected: true,
		},
		{
			name: "egressgateway label on deployment",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-egressgateway",
					Namespace: "istio-system",
					Labels: map[string]string{
						"istio": "egressgateway",
					},
				},
			},
			expected: true,
		},
		{
			name: "istio label with unrelated value",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istiod",
					Namespace: "istio-system",
					Labels: map[string]string{
						"istio": "pilot",
					},
				},
			},
			expected: false,
		},
		{
			name: "no istio label",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-workload",
					Namespace: "default",
					Labels:    map[string]string{},
				},
			},
			expected: false,
		},
		{
			name: "gateway label only on pod template, not on deployment",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-ingressgateway",
					Namespace: "istio-system",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"istio": "ingressgateway",
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIstioGateway(tt.deploy)
			if result != tt.expected {
				t.Errorf("isIstioGateway() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
