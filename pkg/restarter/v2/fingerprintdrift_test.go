package v2

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func deploymentWithStamp(stamp string) client.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"istio-operator.kyma-project.io/restartHash": stamp,
					},
				},
			},
		},
	}
}

func deploymentWithoutStamp() client.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
			},
		},
	}
}

func statefulSetWithStamp(stamp string) client.Object {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-statefulset", Namespace: "default"},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"istio-operator.kyma-project.io/restartHash": stamp,
					},
				},
			},
		},
	}
}

func daemonSetWithStamp(stamp string) client.Object {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-daemonset", Namespace: "default"},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"istio-operator.kyma-project.io/restartHash": stamp,
					},
				},
			},
		},
	}
}

func replicaSetWithStamp(stamp string) client.Object {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-replicaset", Namespace: "default"},
		Spec: appsv1.ReplicaSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"istio-operator.kyma-project.io/restartHash": stamp,
					},
				},
			},
		},
	}
}

func TestFingerprintDriftChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		currentFP      string
		object         client.Object
		expectDecision Decision
	}{
		{
			name:           "empty fingerprint + Deployment with any stamp",
			currentFP:      "",
			object:         deploymentWithStamp("old"),
			expectDecision: Continue,
		},
		{
			name:           "Deployment stamp == fingerprint",
			currentFP:      "same",
			object:         deploymentWithStamp("same"),
			expectDecision: Continue,
		},
		{
			name:           "Deployment stamp != fingerprint",
			currentFP:      "new",
			object:         deploymentWithStamp("old"),
			expectDecision: Restart,
		},
		{
			name:           "Deployment empty stamp + non-empty fingerprint",
			currentFP:      "new",
			object:         deploymentWithoutStamp(),
			expectDecision: Restart,
		},
		{
			name:           "StatefulSet stamp != fingerprint",
			currentFP:      "new",
			object:         statefulSetWithStamp("old"),
			expectDecision: Restart,
		},
		{
			name:           "DaemonSet stamp != fingerprint",
			currentFP:      "new",
			object:         daemonSetWithStamp("old"),
			expectDecision: Restart,
		},
		{
			name:           "ReplicaSet stamp != fingerprint",
			currentFP:      "new",
			object:         replicaSetWithStamp("old"),
			expectDecision: Restart,
		},
		{
			name: "Pod with any annotations",
			currentFP: "new",
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Annotations: map[string]string{
						"istio-operator.kyma-project.io/restartHash": "old",
					},
				},
			},
			expectDecision: Continue,
		},
		{
			name:           "non-workload object (Namespace)",
			currentFP:      "new",
			object:         &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}},
			expectDecision: Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewFingerprintDriftChanged(tt.currentFP)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}
