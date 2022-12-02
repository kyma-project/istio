package pods

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createClientSet(t *testing.T, objects ...client.Object) client.Client {
	err := v1alpha1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()

	return fakeClient
}

func Test_Reset(t *testing.T) {
	ctx := context.Background()

	t.Run("should get a pod with a running sidecar", func(t *testing.T) {
		firstPod := fixPodWithSidecar("app", "enabled", "Running", map[string]string{}, map[string]string{})

		clientSet := createClientSet(t, firstPod)

		pods, err := GetIstioPodsWithSelectors(ctx, clientSet)

		require.NoError(t, err)
		require.Len(t, pods.Items, 1)
	})
}

func fixPodWithSidecar(name, namespace, phase string, annotations map[string]string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []v1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Labels:      labels,
			Annotations: annotations,
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPhase(phase),
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "istio-init",
					Image: "istio-init",
				},
			},
			Containers: []corev1.Container{
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

func fixPodWithoutSidecar(name, namespace, phase string, annotations map[string]string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []v1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
			Labels:      labels,
			Annotations: annotations,
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPhase(phase),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name + "-container",
					Image: "image:6.9",
				},
			},
		},
	}
}
