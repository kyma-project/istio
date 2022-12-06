package pods

import (
	"context"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestGetPodsForCNIChange(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		c             client.Client
		expectedImage SidecarImage
		wantError     bool
		assertFunc    func(t require.TestingT, val interface{})
	}{
		{
			name: "should not get any pod without istio-init container when CNI is enabled",
			c: createClientSet(t,
				fixPodWithoutInitContainer("application1", "enabled", "Running", map[string]string{}, map[string]string{}),
				fixPodWithoutInitContainer("application2", "enabled", "Terminating", map[string]string{}, map[string]string{}),
			),
			wantError:  false,
			assertFunc: func(t require.TestingT, val interface{}) { require.Empty(t, val) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pods, err := getPodsForCNIChange(ctx, tt.c, tt.expectedImage)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			tt.assertFunc(t, pods.Items)
		})
	}
}

func fixPodWithoutInitContainer(name, namespace, phase string, annotations map[string]string, labels map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Labels:      labels,
			Annotations: annotations,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(phase),
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{
				{
					Name:  "istio-validation",
					Image: "istio-validation",
				},
			},
			Containers: []v1.Container{
				{
					Name:  name + "-container",
					Image: "image:6.9",
				},
				{
					Name:  "istio-proxy",
					Image: "istio-proxy",
				},
			},
		},
	}
}
