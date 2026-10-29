package v2

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGatewayFingerprintDriftChanged_Evaluate(t *testing.T) {
	tests := []struct {
		name               string
		currentFingerprint string
		obj                Object
		expected           Decision
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
								fingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: Restart,
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
								fingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: Restart,
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
								fingerprintAnnotationKey: "hash-v1",
							},
						},
					},
				},
			},
			expected: Continue,
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
								fingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: Continue,
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
								fingerprintAnnotationKey: "hash-v2",
							},
						},
					},
				},
			},
			expected: Continue,
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
			expected: Restart,
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
						fingerprintAnnotationKey: "hash-v2",
					},
				},
			},
			expected: Continue,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evaluator := NewGatewayFingerprintDriftChanged(tc.currentFingerprint)
			result := evaluator.Evaluate(tc.obj)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
