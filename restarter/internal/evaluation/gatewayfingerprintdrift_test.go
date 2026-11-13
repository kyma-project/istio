package evaluation_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGatewayFingerprintChanged_Evaluate(t *testing.T) {
	tests := []struct {
		name               string
		currentFingerprint string
		obj                evaluation.Object
		expected           evaluation.Decision
	}{
		{
			name:               "ingress gateway with differing stamp returns Restart",
			currentFingerprint: "hash-v1",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-ingressgateway",
					Namespace: "istio-system",
					Labels:    map[string]string{"istio": "ingressgateway"},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								evaluation.FingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: evaluation.Restart,
		},
		{
			name:               "egress gateway with differing stamp returns Restart",
			currentFingerprint: "hash-v1",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-egressgateway",
					Namespace: "istio-system",
					Labels:    map[string]string{"istio": "egressgateway"},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								evaluation.FingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: evaluation.Restart,
		},
		{
			name:               "gateway with matching stamp returns Continue",
			currentFingerprint: "hash-v1",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-ingressgateway",
					Namespace: "istio-system",
					Labels:    map[string]string{"istio": "ingressgateway"},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								evaluation.FingerprintAnnotationKey: "hash-v1",
							},
						},
					},
				},
			},
			expected: evaluation.Continue,
		},
		{
			name:               "empty current fingerprint returns Continue",
			currentFingerprint: "",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-ingressgateway",
					Namespace: "istio-system",
					Labels:    map[string]string{"istio": "ingressgateway"},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								evaluation.FingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: evaluation.Continue,
		},
		{
			name:               "non-gateway deployment with differing stamp returns Continue",
			currentFingerprint: "hash-v1",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-other-deployment",
					Namespace: "istio-system",
					Labels:    map[string]string{"istio": "pilot"},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								evaluation.FingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: evaluation.Continue,
		},
		{
			name:               "gateway with no stamp annotation returns Restart",
			currentFingerprint: "hash-v1",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-ingressgateway",
					Namespace: "istio-system",
					Labels:    map[string]string{"istio": "ingressgateway"},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{},
				},
			},
			expected: evaluation.Restart,
		},
		{
			name:               "pod returns Continue",
			currentFingerprint: "hash-v1",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-pod",
					Namespace: "istio-system",
					Labels:    map[string]string{"istio": "ingressgateway"},
					Annotations: map[string]string{
						evaluation.FingerprintAnnotationKey: "hash-v2",
					},
				},
			},
			expected: evaluation.Continue,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evaluator := evaluation.NewGatewayFingerprintChanged(tc.currentFingerprint)
			result := evaluator.Evaluate(tc.obj)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
